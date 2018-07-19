package peer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/stellar/go/xdr"
)

type PeerContext struct {
	Conn                net.Conn
	sendMessageSequence xdr.Uint64
	CachedAuthCert      xdr.AuthCert

	AuthSecretKey [32]byte
	AuthPublicKey [32]byte
	AuthSharedKey []byte

	ReceivingMacKey []byte
	SendingMacKey   []byte

	LocalNonce [32]byte
}

func Connect(address string) (*PeerContext, error) {
	// Connect to validator
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	context := PeerContext{
		Conn: conn,
	}

	return &context, nil
}

func (context *PeerContext) SendMessage(message xdr.StellarMessage) {
	am0 := xdr.AuthenticatedMessageV0{
		Sequence: context.sendMessageSequence,
		Message:  message,
	}

	if message.Type != xdr.MessageTypeHello {
		buf := bytes.Buffer{}
		xdr.Marshal(&buf, &am0.Sequence)
		xdr.Marshal(&buf, &am0.Message)
		hmac := hmac.New(sha256.New, context.SendingMacKey)
		hmac.Write(buf.Bytes())
		var mac [32]byte
		copy(mac[:], hmac.Sum(nil))
		am0.Mac = xdr.HmacSha256Mac{Mac: mac}
		context.sendMessageSequence++
	}

	am, _ := xdr.NewAuthenticatedMessage(xdr.Uint32(0), am0)

	var messageBuffer bytes.Buffer
	xdr.Marshal(&messageBuffer, &am)
	outBytes := messageBuffer.Bytes()
	context.sendHeader(uint32(len(outBytes)))
	context.Conn.Write(messageBuffer.Bytes())
}

func (context *PeerContext) ReceiveMessage() xdr.StellarMessage {
	length := context.receiveHeader()
	if length <= 0 {
		fmt.Println("Got a length of 0 or smaller")
	}

	buf := make([]byte, 0, length)
	bytesRead := 0
	for {
		tmp := make([]byte, length-bytesRead)
		n, err := context.Conn.Read(tmp)
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

func (context *PeerContext) sendHeader(length uint32) {
	// In RPC (see RFC5531 section 11), the high bit means this is the
	// last record fragment in a record.  If the high bit is clear, it
	// means another fragment follows.  We don't currently implement
	// continuation fragments, and instead always set the last-record
	// bit to produce a single-fragment record.

	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, length|0x80000000)
	context.Conn.Write(header)
}

func (context *PeerContext) receiveHeader() int {
	header := make([]byte, 4)
	read, err := context.Conn.Read(header)

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
