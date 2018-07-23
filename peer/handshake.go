package peer

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/stellar/go/xdr"
	"github.com/tinco/stellar-core-go/nodeInfo"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ed25519"
)

func (peer *Peer) setupCrypto() {
	rand.Read(peer.localNonce[:])
	rand.Read(peer.authSecretKey[:])
	// Set up auth public key
	curve25519.ScalarBaseMult(&peer.authPublicKey, &peer.authSecretKey)
}

func (peer *Peer) startAuthentication(nodeInfo *nodeInfo.NodeInfo) {
	peer.setupCrypto()

	peerID, err := xdr.NewNodeId(xdr.PublicKeyTypePublicKeyTypeEd25519, nodeInfo.PublicKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	authCert := peer.getAuthCert(nodeInfo)

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
		Nonce:             xdr.Uint256(peer.localNonce),
	}

	message, err := xdr.NewStellarMessage(xdr.MessageTypeHello, hello)
	if err != nil {
		fmt.Println(err)
	}

	peer.sendMessage(message)

	peer.MustRespond()
	helloResponse := peer.receiveMessage().MustHello()
	peer.handleHello(helloResponse)

	// Auth is just an empty message with a valid mac
	auth := xdr.Auth{}

	message, err = xdr.NewStellarMessage(xdr.MessageTypeAuth, auth)
	if err != nil {
		fmt.Println(err)
	}

	peer.sendMessage(message)

	peer.MustRespond()
	peer.receiveMessage().MustAuth()
}

func (peer *Peer) handleHello(hello xdr.Hello) {
	remotePublicKey := hello.Cert.Pubkey
	remoteNonce := hello.Nonce
	peer.setupRemoteKeys(remotePublicKey.Key, remoteNonce, true)
}

func (peer *Peer) setupRemoteKeys(remotePublicKey [32]byte, remoteNonce [32]byte, weCalled bool) {
	// Set up auth shared key
	var publicA [32]byte
	var publicB [32]byte

	if weCalled {
		publicA = peer.authPublicKey
		publicB = remotePublicKey
	} else {
		publicA = remotePublicKey
		publicB = peer.authPublicKey
	}

	var q [32]byte
	curve25519.ScalarMult(&q, &peer.authSecretKey, &remotePublicKey)

	buf := bytes.NewBuffer(q[:])
	buf.Write(publicA[:])
	buf.Write(publicB[:])

	peer.authSharedKey = hkdfExtract(buf.Bytes())

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
	buf.Write(peer.localNonce[:])
	buf.Write(remoteNonce[:])

	peer.sendingMacKey = hkdfExpand(peer.authSharedKey, buf)

	// Set up receivingMacKey
	buf = &bytes.Buffer{}

	if weCalled {
		buf.WriteByte(0)
	} else {
		buf.WriteByte(1)
	}
	buf.Write(remoteNonce[:])
	buf.Write(peer.localNonce[:])

	peer.receivingMacKey = hkdfExpand(peer.authSharedKey, buf)
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

func (peer *Peer) getAuthCert(nodeInfo *nodeInfo.NodeInfo) xdr.AuthCert {
	now := time.Now().Unix()

	if peer.cachedAuthCert.Expiration > xdr.Uint64(now) {
		return peer.cachedAuthCert
	}

	expirationLimit := int64(3600) // one hour
	expiration := xdr.Uint64(now + expirationLimit)

	var messageDataBuffer bytes.Buffer

	xdr.Marshal(&messageDataBuffer, &nodeInfo.NetworkID)
	xdr.Marshal(&messageDataBuffer, xdr.EnvelopeTypeEnvelopeTypeAuth)
	xdr.Marshal(&messageDataBuffer, &expiration)
	xdr.Marshal(&messageDataBuffer, &peer.authPublicKey)

	hash := sha256.Sum256(messageDataBuffer.Bytes())
	sig := sign(nodeInfo, hash)

	peer.cachedAuthCert = xdr.AuthCert{
		Pubkey:     xdr.Curve25519Public{Key: peer.authPublicKey},
		Expiration: xdr.Uint64(expiration),
		Sig:        sig,
	}

	return peer.cachedAuthCert
}
