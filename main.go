package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"time"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ed25519"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

const secretSeedString string = "SAN6S4HURKTECO6MGKDKNPQUZFEDDW7CODR63ZIEKGFW27MUWZX2TNV2"
const publicKeyString string = "GCFVEVUGA62TM3P2HCBZRRAGIV4CMDNZGQYXD733LSEO6RDHPU5H7MOX"
const networkPassPhrase string = "Public Global Stellar Network ; September 2015"

var secretSeedBytes []byte
var publicKeyBytes []byte
var secretSeed xdr.Uint256
var publicKey xdr.Uint256
var privateKey ed25519.PrivateKey

var cachedAuthCert xdr.AuthCert
var networkID xdr.Hash

var sendMessageSequence xdr.Uint64

var authSecretKey [32]byte
var authPublicKey [32]byte
var authSharedKey []byte

var receivingMacKey []byte
var sendingMacKey []byte

var localNonce [32]byte

func setupCrypto() {
	var err error
	secretSeedBytes, err = strkey.Decode(strkey.VersionByteSeed, secretSeedString)
	if err != nil {
		fmt.Println(err)
		panic("Could not initialize keys.")
	}

	publicKeyBytes, err = strkey.Decode(strkey.VersionByteAccountID, publicKeyString)

	copy(secretSeed[:], secretSeedBytes)
	copy(publicKey[:], publicKeyBytes)

	networkID = xdr.Hash(sha256.Sum256([]byte(networkPassPhrase)))
	privateKey = ed25519.NewKeyFromSeed(secretSeedBytes)
	rand.Read(localNonce[:])

	if err != nil {
		fmt.Println(err)
		panic("Could not initialize keys.")
	}

	// Set up auth secret key
	rand.Read(authSecretKey[:])
	fmt.Printf("authSecretKey: %s\n", hex.EncodeToString(authSecretKey[:]))

	// Set up auth public key
	curve25519.ScalarBaseMult(&authPublicKey, &authSecretKey)
	fmt.Printf("authPublicKey: %s\n", hex.EncodeToString(authPublicKey[:]))
}

func handleHello(hello xdr.Hello) {
	remotePublicKey := hello.Cert.Pubkey
	remoteNonce := hello.Nonce
	setupRemoteKeys(remotePublicKey.Key, remoteNonce, true)
}

func setupRemoteKeys(remotePublicKey [32]byte, remoteNonce [32]byte, weCalled bool) {
	fmt.Printf("remotePublicKey: %s\n", hex.EncodeToString(remotePublicKey[:]))

	// Set up auth shared key
	var publicA [32]byte
	var publicB [32]byte

	if weCalled {
		publicA = authPublicKey
		publicB = remotePublicKey
	} else {
		publicA = remotePublicKey
		publicB = authPublicKey
	}

	var q [32]byte
	curve25519.ScalarMult(&q, &authSecretKey, &remotePublicKey)

	buf := bytes.NewBuffer(q[:])
	buf.Write(publicA[:])
	buf.Write(publicB[:])

	authSharedKey = hkdfExtract(q[:])
	fmt.Printf("authSharedKey: %s\n", hex.EncodeToString(authSharedKey[:]))

	// Set up sendingMacKey

	// If weCalled then sending key is K_AB,
	// and A is local and B is remote.
	// If REMOTE_CALLED_US then sending key is K_BA,
	// and B is local and A is remote.

	buf = &bytes.Buffer{}
	if weCalled {
		buf.WriteByte(0)
	} else {
		buf.WriteByte(1)
	}
	buf.Write(localNonce[:])
	buf.Write(remoteNonce[:])

	sendingMacKey = hkdfExpand(authSharedKey, buf)
	fmt.Printf("sendingMacKey: %s\n", hex.EncodeToString(sendingMacKey[:]))

	// Set up receivingMacKey

	buf = &bytes.Buffer{}

	if weCalled {
		buf.WriteByte(0)
	} else {
		buf.WriteByte(1)
	}
	buf.Write(remoteNonce[:])
	buf.Write(localNonce[:])

	receivingMacKey = hkdfExpand(authSharedKey, buf)
	fmt.Printf("receivingMacKey: %s\n", hex.EncodeToString(receivingMacKey[:]))

}

func hkdfExtract(buf []byte) []byte {
	zerosalt := make([]byte, 32)
	hmac := hmac.New(sha256.New, zerosalt)
	hmac.Write(buf)
	return hmac.Sum(nil)
}

func hkdfExpand(key []byte, buf *bytes.Buffer) []byte {
	buf.WriteByte(1)
	hmac := hmac.New(sha256.New, key)
	hmac.Write(buf.Bytes())
	return hmac.Sum(nil)
}

func main() {
	fmt.Println("Stellar Go Debug Client")

	setupCrypto()

	// Connect to validator
	// conn, err := net.Dial("tcp", "stellar0.keybase.io:11625")
	conn, err := net.Dial("tcp", "localhost:11625")
	if err != nil {
		fmt.Println(err)
		return
	}

	if err != nil {
		fmt.Println(err)
		return
	}

	peerID, err := xdr.NewNodeId(xdr.PublicKeyTypePublicKeyTypeEd25519, xdr.Uint256(publicKey))
	if err != nil {
		fmt.Println(err)
		return
	}

	authCert := getAuthCert()

	// Send Hello message
	hello := xdr.Hello{
		LedgerVersion:     9000,
		OverlayVersion:    9000,
		OverlayMinVersion: 0,
		NetworkId:         networkID,
		VersionStr:        "stellar-core-go[alpha-0.0]",
		ListeningPort:     11625,
		PeerId:            peerID,
		Cert:              authCert,
		Nonce:             xdr.Uint256(localNonce),
	}

	message, err := xdr.NewStellarMessage(xdr.MessageTypeHello, hello)
	if err != nil {
		fmt.Println(err)
	}

	sendMessage(conn, message)

	helloResponse := receiveMessage(conn).MustHello()
	handleHello(helloResponse)

	// Print any responses
	fmt.Printf("response: %+v\n\n", helloResponse)

	// Auth is just an empty message with a valid mac
	auth := xdr.Auth{}

	message, err = xdr.NewStellarMessage(xdr.MessageTypeAuth, auth)
	if err != nil {
		fmt.Println(err)
	}

	sendMessage(conn, message)

	authResponse := receiveMessage(conn).MustError()
	fmt.Printf("response: %+v", authResponse)

}

func sign(hash [sha256.Size]byte) xdr.Signature {
	signature := ed25519.Sign(privateKey, hash[:])
	return xdr.Signature(signature)
}

func getAuthCert() xdr.AuthCert {
	now := time.Now().Unix()

	if cachedAuthCert.Expiration > xdr.Uint64(now) {
		return cachedAuthCert
	}

	expirationLimit := int64(3600) // one hour
	expiration := xdr.Uint64(now + expirationLimit)

	var messageDataBuffer bytes.Buffer

	xdr.Marshal(&messageDataBuffer, &networkID)
	xdr.Marshal(&messageDataBuffer, xdr.EnvelopeTypeEnvelopeTypeAuth)
	xdr.Marshal(&messageDataBuffer, &expiration)
	xdr.Marshal(&messageDataBuffer, &authPublicKey)

	// fmt.Printf("AuthCertBytes: %s", hex.Dump(messageDataBuffer.Bytes()))

	hash := sha256.Sum256(messageDataBuffer.Bytes())
	sig := sign(hash)

	// fmt.Printf("Hash: %s", hex.Dump(hash[:]))
	// fmt.Printf("Sig: %s", hex.Dump(sig))

	cachedAuthCert = xdr.AuthCert{
		Pubkey:     xdr.Curve25519Public{Key: authPublicKey},
		Expiration: xdr.Uint64(expiration),
		Sig:        sig,
	}

	return cachedAuthCert

}

func sendMessage(conn net.Conn, message xdr.StellarMessage) {
	am0 := xdr.AuthenticatedMessageV0{
		Sequence: sendMessageSequence,
		Message:  message,
	}

	if message.Type != xdr.MessageTypeHello {
		buf := bytes.Buffer{}
		xdr.Marshal(&buf, &am0.Sequence)
		xdr.Marshal(&buf, &am0.Message)
		hmac := hmac.New(sha256.New, sendingMacKey)
		hmac.Write(buf.Bytes())
		var mac [32]byte
		copy(mac[:], hmac.Sum(nil))
		am0.Mac = xdr.HmacSha256Mac{Mac: mac}

		fmt.Printf("Mac: %s\n", hex.EncodeToString(mac[:]))
		fmt.Printf("MacKey: %s\n", hex.EncodeToString(sendingMacKey))
		fmt.Printf("Msg: %s\n", hex.EncodeToString(buf.Bytes()))

		/* It expects that the message mac, with the received mac key, together verify the sequence + message */

		sendMessageSequence++
	}

	am, _ := xdr.NewAuthenticatedMessage(xdr.Uint32(0), am0)

	var messageBuffer bytes.Buffer
	xdr.Marshal(&messageBuffer, &am)
	outBytes := messageBuffer.Bytes()
	sendHeader(conn, uint32(len(outBytes)))
	conn.Write(messageBuffer.Bytes())
}

func receiveMessage(conn net.Conn) xdr.StellarMessage {
	length := receiveHeader(conn)
	if length <= 0 {
		fmt.Println("Got a length of 0 or smaller")
	}

	buf := make([]byte, 0, length)
	bytesRead := 0
	for {
		tmp := make([]byte, length-bytesRead)
		n, err := conn.Read(tmp)
		bytesRead += n
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error:", err)
			}
			break
		}
		buf = append(buf, tmp[:n]...)

		if bytesRead >= length {
			break
		}
	}

	var message xdr.AuthenticatedMessage
	_, err := xdr.Unmarshal(bytes.NewReader(buf), &message)
	if err != nil {
		fmt.Println(err)
	}

	// fmt.Printf("Buffer : %v\n", buf)

	return message.MustV0().Message
}

func sendHeader(conn net.Conn, length uint32) {
	// In RPC (see RFC5531 section 11), the high bit means this is the
	// last record fragment in a record.  If the high bit is clear, it
	// means another fragment follows.  We don't currently implement
	// continuation fragments, and instead always set the last-record
	// bit to produce a single-fragment record.

	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, length|0x80000000)
	conn.Write(header)
}

func receiveHeader(conn net.Conn) int {
	header := make([]byte, 4)
	read, err := conn.Read(header)

	if err != nil {
		fmt.Println(err)
	}

	if read != 4 {
		fmt.Println("Tried to receive header, but didn't get 4 bytes", read)
		panic("Receive Header failed")
	}

	length := 0
	length = int(header[0])
	length &= 0x7f // clear the XDR 'continuation' bit
	length <<= 8
	length |= int(header[1])
	length <<= 8
	length |= int(header[2])
	length <<= 8
	length |= int(header[3])
	return length
}
