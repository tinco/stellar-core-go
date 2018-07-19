package handshake

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/stellar/go/xdr"
	"github.com/tinco/stellar-core-go/nodeInfo"
	"github.com/tinco/stellar-core-go/peer"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ed25519"
)

func setupCrypto(context *peer.PeerContext) {
	rand.Read(context.LocalNonce[:])
	rand.Read(context.AuthSecretKey[:])
	// Set up auth public key
	curve25519.ScalarBaseMult(&context.AuthPublicKey, &context.AuthSecretKey)
}

func StartAuthentication(nodeInfo *nodeInfo.NodeInfo, context *peer.PeerContext) {
	setupCrypto(context)

	peerID, err := xdr.NewNodeId(xdr.PublicKeyTypePublicKeyTypeEd25519, nodeInfo.PublicKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	authCert := getAuthCert(nodeInfo, context)

	// Send Hello message
	hello := xdr.Hello{
		LedgerVersion:     9000,
		OverlayVersion:    9000,
		OverlayMinVersion: 0,
		NetworkId:         nodeInfo.NetworkID,
		VersionStr:        "stellar-core-go[alpha-0.0]",
		ListeningPort:     11625,
		PeerId:            peerID,
		Cert:              authCert,
		Nonce:             xdr.Uint256(context.LocalNonce),
	}

	message, err := xdr.NewStellarMessage(xdr.MessageTypeHello, hello)
	if err != nil {
		fmt.Println(err)
	}

	context.SendMessage(message)

	helloResponse := context.ReceiveMessage().MustHello()
	handleHello(context, helloResponse)

	// Print any responses
	fmt.Printf("response: %+v\n\n", helloResponse)

	// Auth is just an empty message with a valid mac
	auth := xdr.Auth{}

	message, err = xdr.NewStellarMessage(xdr.MessageTypeAuth, auth)
	if err != nil {
		fmt.Println(err)
	}

	context.SendMessage(message)

	authResponse := context.ReceiveMessage()
	fmt.Printf("response: %+v", authResponse)
}

func handleHello(context *peer.PeerContext, hello xdr.Hello) {
	remotePublicKey := hello.Cert.Pubkey
	remoteNonce := hello.Nonce
	setupRemoteKeys(context, remotePublicKey.Key, remoteNonce, true)
}

func setupRemoteKeys(context *peer.PeerContext, remotePublicKey [32]byte, remoteNonce [32]byte, weCalled bool) {
	// fmt.Printf("remotePublicKey: %s\n", hex.EncodeToString(remotePublicKey[:]))

	// Set up auth shared key
	var publicA [32]byte
	var publicB [32]byte

	if weCalled {
		publicA = context.AuthPublicKey
		publicB = remotePublicKey
	} else {
		publicA = remotePublicKey
		publicB = context.AuthPublicKey
	}

	var q [32]byte
	curve25519.ScalarMult(&q, &context.AuthSecretKey, &remotePublicKey)

	buf := bytes.NewBuffer(q[:])
	buf.Write(publicA[:])
	buf.Write(publicB[:])

	context.AuthSharedKey = hkdfExtract(buf.Bytes())

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
	buf.Write(context.LocalNonce[:])
	buf.Write(remoteNonce[:])

	context.SendingMacKey = hkdfExpand(context.AuthSharedKey, buf)

	// Set up receivingMacKey
	buf = &bytes.Buffer{}

	if weCalled {
		buf.WriteByte(0)
	} else {
		buf.WriteByte(1)
	}
	buf.Write(remoteNonce[:])
	buf.Write(context.LocalNonce[:])

	context.ReceivingMacKey = hkdfExpand(context.AuthSharedKey, buf)
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

func sign(nodeInfo *nodeInfo.NodeInfo, hash [sha256.Size]byte) xdr.Signature {
	signature := ed25519.Sign(nodeInfo.PrivateKey, hash[:])
	return xdr.Signature(signature)
}

func getAuthCert(nodeInfo *nodeInfo.NodeInfo, context *peer.PeerContext) xdr.AuthCert {
	now := time.Now().Unix()

	if context.CachedAuthCert.Expiration > xdr.Uint64(now) {
		return context.CachedAuthCert
	}

	expirationLimit := int64(3600) // one hour
	expiration := xdr.Uint64(now + expirationLimit)

	var messageDataBuffer bytes.Buffer

	xdr.Marshal(&messageDataBuffer, &nodeInfo.NetworkID)
	xdr.Marshal(&messageDataBuffer, xdr.EnvelopeTypeEnvelopeTypeAuth)
	xdr.Marshal(&messageDataBuffer, &expiration)
	xdr.Marshal(&messageDataBuffer, &context.AuthPublicKey)

	// fmt.Printf("AuthCertBytes: %s", hex.Dump(messageDataBuffer.Bytes()))

	hash := sha256.Sum256(messageDataBuffer.Bytes())
	sig := sign(nodeInfo, hash)

	// fmt.Printf("Hash: %s", hex.Dump(hash[:]))
	// fmt.Printf("Sig: %s", hex.Dump(sig))

	context.CachedAuthCert = xdr.AuthCert{
		Pubkey:     xdr.Curve25519Public{Key: context.AuthPublicKey},
		Expiration: xdr.Uint64(expiration),
		Sig:        sig,
	}

	return context.CachedAuthCert
}
