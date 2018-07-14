package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"reflect"
	"runtime"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
)

// MessageType represents the type of the SCP message
type MessageType int

const (
	// ErrorMsg is a message that conveys an error has happened at the sender
	ErrorMsg MessageType = 0

	// Auth is a message that indicates the sender wants to authenticate
	Auth MessageType = 2

	// DontHave (indicates the sender doesn't have something?)
	DontHave MessageType = 3

	// GetPeers requests the recipient responds with a list of peers
	GetPeers MessageType = 4

	// Peers is a message with the list of peers the sender is connected to
	Peers MessageType = 5

	// GetTxSet requests the sender responds with a transaction set identified by a hash
	GetTxSet MessageType = 6

	// TxSet is a message that contains a transaction set requested previously by recipient
	TxSet MessageType = 7

	// Transaction has details about a transaction that a peer has heard about
	Transaction MessageType = 8

	// GetScpQuorumset requests the recipient to respond with its quorum set
	GetScpQuorumset MessageType = 9

	// ScpQuorumset informs the recipient of the quorumset the sender is part of
	ScpQuorumset MessageType = 10

	// ScpMessage is a general message regarding SCP (??)
	ScpMessage MessageType = 11

	//GetScpState requests the sender respond with SCP status (??) (should be this be ScpState?)
	GetScpState MessageType = 12

	// Hello introduces recipient of the intent to communicate (necessary before Auth)
	Hello MessageType = 13
)

func main() {
	fmt.Println("Stellar Go Debug Client")
	// Connect to validator
	// conn, err := net.Dial("tcp", "stellar0.keybase.io:11625")
	conn, err := net.Dial("tcp", "localhost:11625")
	if err != nil {
		fmt.Println(err)
		return
	}

	// defer stackDump(&err, main)

	// secretSeedString := "SAN6S4HURKTECO6MGKDKNPQUZFEDDW7CODR63ZIEKGFW27MUWZX2TNV2"
	publicKeyString := "GCFVEVUGA62TM3P2HCBZRRAGIV4CMDNZGQYXD733LSEO6RDHPU5H7MOX"

	// secretKey, err := strkey.Decode(strkey.VersionByteSeed, secretSeedString)
	publicKey, err := strkey.Decode(strkey.VersionByteAccountID, publicKeyString)
	var publicKeyBytes xdr.Uint256
	copy(publicKeyBytes[:], publicKey)

	if err != nil {
		fmt.Println(err)
		return
	}

	networkID := xdr.Hash(sha256.Sum256([]byte("mysimpleclient-1")))
	peerID, err := xdr.NewNodeId(xdr.PublicKeyTypePublicKeyTypeEd25519, publicKeyBytes)
	if err != nil {
		fmt.Println(err)
		return
	}

	authCert := xdr.AuthCert{}

	// Send Hello message
	hello := xdr.Hello{
		LedgerVersion:     9000,
		OverlayVersion:    9000,
		OverlayMinVersion: 0,
		NetworkId:         networkID,
		VersionStr:        "stellar-core-go[alpha-0.0]",
		ListeningPort:     0,
		PeerId:            peerID,
		Cert:              authCert,
		Nonce:             xdr.Uint256{1},
	}

	message, err := xdr.NewStellarMessage(xdr.MessageTypeHello, hello)
	if err != nil {
		fmt.Println(err)
	}

	sendMessage(conn, publicKey, message)

	response := receiveMessage(conn)

	error := response.MustError()
	fmt.Printf("Error: %+v\n", error.Msg)

	// Print any responses
	fmt.Printf("response: %+v", response)
}

func sendMessage(conn net.Conn, key []byte, message xdr.StellarMessage) {
	//mac := hmac.New(sha256.New, key)
	// mac.Sum()
	var mac [32]byte
	am0 := xdr.AuthenticatedMessageV0{
		Sequence: xdr.Uint64(0),
		Message:  message,
		Mac:      xdr.HmacSha256Mac{Mac: mac},
	}
	am, _ := xdr.NewAuthenticatedMessage(xdr.Uint32(0), am0)

	fmt.Printf("AM : %v\n", am)
	fmt.Printf("AM0 : %v\n", am0)

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

	buf := make([]byte, 0, length) // big buffer
	bytesRead := 0
	for {
		tmp := make([]byte, length-bytesRead) // using small tmo buffer for demonstrating
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

	fmt.Println("got", bytesRead, "bytes.")

	var message xdr.StellarMessage
	bytesRead, err := xdr.Unmarshal(bytes.NewReader(buf), &message)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("bytes read:", bytesRead)
	fmt.Println("length expected:", length)
	fmt.Printf("Buffer : %v\n", buf)

	return message //.MustV0().Message
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

func stackDump(err *error, f interface{}) {
	fname := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	_, file, line, _ := runtime.Caller(4) // this skips the first 4 that are called under log.Panic()
	if r := recover(); r != nil {
		fmt.Printf("%s (recover): %v\n", fname, r)
		if err != nil {
			*err = fmt.Errorf("%v", r)
		}
	} else if err != nil && *err != nil {
		fmt.Printf("%s : %v\n", fname, *err)
	}

	buf := make([]byte, 1<<10)
	runtime.Stack(buf, false)
	fmt.Println("==> stack trace: [PANIC:", file, line, fname+"]")
	fmt.Println(string(buf))
}
