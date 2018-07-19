package peer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/stellar/go/xdr"
	"github.com/tinco/stellar-core-go/nodeInfo"
)

type listener func(xdr.StellarMessage)

// Peer represents a connection to a peer
type Peer struct {
	Conn                net.Conn
	sendMessageSequence xdr.Uint64
	cachedAuthCert      xdr.AuthCert

	authSecretKey [32]byte
	authPublicKey [32]byte
	authSharedKey []byte

	receivingMacKey []byte
	sendingMacKey   []byte

	localNonce [32]byte

	listeners map[xdr.MessageType]listener
}

// Connect to validator
func Connect(nodeInfo *nodeInfo.NodeInfo, address string) (*Peer, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	peer := Peer{
		Conn:      conn,
		listeners: make(map[xdr.MessageType]listener),
	}

	peer.startAuthentication(nodeInfo)

	go peer.listen()

	return &peer, nil
}

func (peer *Peer) listen() {
	for {
		message := peer.receiveMessage()
		listener, ok := peer.listeners[message.Type]
		if ok {
			go listener(message)
		} else {
			// fmt.Printf("Received unsollicited message: %v\n\n", message)
		}
	}
}

func (peer *Peer) waitForMessage(typ xdr.MessageType) (*xdr.StellarMessage, error) {
	messageChan := make(chan xdr.StellarMessage)
	peer.listeners[typ] = func(message xdr.StellarMessage) {
		// this is a race condition, subscribing from multiple goroutines is dangerous
		delete(peer.listeners, message.Type)
		messageChan <- message
	}

	select {
	case message := <-messageChan:
		return &message, nil
	case <-time.After(5 * time.Second):
		return nil, errors.New("Waiting for message timed out")
	}
}

// WaitForMessages listens for messages of the given type, returning a channel
// that the messages are put on, as well as a channel that is used to indicate
// we are done listening for messages.
func (peer *Peer) WaitForMessages(typ xdr.MessageType) (chan xdr.StellarMessage, chan struct{}) {
	messageChan := make(chan xdr.StellarMessage)
	peer.listeners[typ] = func(message xdr.StellarMessage) {
		messageChan <- message
	}

	doneChan := make(chan struct{})

	go func() {
		<-doneChan
		// this is a race condition, subscribing from multiple goroutines is dangerous
		delete(peer.listeners, typ)
	}()

	return messageChan, doneChan
}

func (peer *Peer) sendMessage(message xdr.StellarMessage) {
	am0 := xdr.AuthenticatedMessageV0{
		Sequence: peer.sendMessageSequence,
		Message:  message,
	}

	if message.Type != xdr.MessageTypeHello {
		buf := bytes.Buffer{}
		xdr.Marshal(&buf, &am0.Sequence)
		xdr.Marshal(&buf, &am0.Message)
		hmac := hmac.New(sha256.New, peer.sendingMacKey)
		hmac.Write(buf.Bytes())
		var mac [32]byte
		copy(mac[:], hmac.Sum(nil))
		am0.Mac = xdr.HmacSha256Mac{Mac: mac}
		peer.sendMessageSequence++
	}

	am, _ := xdr.NewAuthenticatedMessage(xdr.Uint32(0), am0)

	var messageBuffer bytes.Buffer
	xdr.Marshal(&messageBuffer, &am)
	outBytes := messageBuffer.Bytes()
	peer.sendHeader(uint32(len(outBytes)))
	peer.Conn.Write(messageBuffer.Bytes())
}

// Don't use for anything else than the handshake, as we can receive messages
// out of band, use waitForMessage instead.
func (peer *Peer) receiveMessage() xdr.StellarMessage {
	length := peer.receiveHeader()
	if length <= 0 {
		fmt.Println("Got a length of 0 or smaller")
	}

	buf := make([]byte, 0, length)
	bytesRead := 0
	for {
		tmp := make([]byte, length-bytesRead)
		n, err := peer.Conn.Read(tmp)
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

func (peer *Peer) sendHeader(length uint32) {
	// In RPC (see RFC5531 section 11), the high bit means this is the
	// last record fragment in a record.  If the high bit is clear, it
	// means another fragment follows.  We don't currently implement
	// continuation fragments, and instead always set the last-record
	// bit to produce a single-fragment record.

	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, length|0x80000000)
	peer.Conn.Write(header)
}

func (peer *Peer) receiveHeader() int {
	header := make([]byte, 4)
	read, err := peer.Conn.Read(header)

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
