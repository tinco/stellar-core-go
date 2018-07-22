package peer

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

// GetPeerAddresses gets a list of peers
func (peer *Peer) GetPeerAddresses() {
	command := 0
	message, err := xdr.NewStellarMessage(xdr.MessageTypeGetPeers, command)
	if err != nil {
		fmt.Println(err)
	}

	peer.sendMessage(message)
}

// GetTxSet gets the transaction set
func (peer *Peer) GetTxSet(hash xdr.Hash) {
	command := xdr.Uint256(hash)
	message, err := xdr.NewStellarMessage(xdr.MessageTypeGetTxSet, command)
	if err != nil {
		fmt.Println(err)
	}

	peer.sendMessage(message)
}

// AnnounceTransaction informs peer of a transaction
func (peer *Peer) AnnounceTransaction(tx xdr.Transaction) {
	command := tx
	message, err := xdr.NewStellarMessage(xdr.MessageTypeTransaction, command)
	if err != nil {
		fmt.Println(err)
	}

	peer.sendMessage(message)
}

// GetScpQuorumset gets scp quorum set
func (peer *Peer) GetScpQuorumset(hash xdr.Hash) {
	command := xdr.Uint256(hash)
	message, err := xdr.NewStellarMessage(xdr.MessageTypeGetScpQuorumset, command)
	if err != nil {
		fmt.Println(err)
		panic("Omg did something wrong in making get quorumset")
	}
	peer.sendMessage(message)
}

// GetScpState gets the scp state
func (peer *Peer) GetScpState() {
	command := 0
	message, err := xdr.NewStellarMessage(xdr.MessageTypeGetScpState, command)
	if err != nil {
		fmt.Println(err)
	}

	peer.sendMessage(message)
}
