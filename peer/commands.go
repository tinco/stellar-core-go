package peer

import (
	"fmt"
	"net"
	"strconv"

	"github.com/stellar/go/xdr"
)

// GetPeerAddresses gets a list of peers
func (peer *Peer) GetPeerAddresses() []string {
	command := 0
	message, err := xdr.NewStellarMessage(xdr.MessageTypeGetPeers, command)
	if err != nil {
		fmt.Println(err)
	}

	peer.sendMessage(message)
	m, _ := peer.waitForMessage(xdr.MessageTypePeers)
	peers := m.MustPeers()
	peerAddresses := make([]string, len(peers))
	for i, v := range peers {
		var ipBytes []byte
		if v.Ip.Type.String() == "IpAddrTypeIPv4" {
			bytes := v.Ip.MustIpv4()
			ipBytes = bytes[:]
		} else {
			bytes := v.Ip.MustIpv6()
			ipBytes = bytes[:]
		}
		ip := net.IP(ipBytes).String()
		peerAddresses[i] = ip + ":" + strconv.FormatUint(uint64(v.Port), 10)
	}

	return peerAddresses
}

// GetTxSet gets the transaction set
func (peer *Peer) GetTxSet() xdr.TransactionSet {
	command := 0
	message, err := xdr.NewStellarMessage(xdr.MessageTypeGetTxSet, command)
	if err != nil {
		fmt.Println(err)
	}

	peer.sendMessage(message)
	txset, _ := peer.waitForMessage(xdr.MessageTypeTxSet)
	return txset.MustTxSet()
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
func (peer *Peer) GetScpQuorumset() xdr.ScpQuorumSet {
	command := xdr.Uint256{0}
	message, err := xdr.NewStellarMessage(xdr.MessageTypeGetScpQuorumset, command)
	if err != nil {
		fmt.Println(err)
		panic("Omg did something wrong in making get quorumset")
	}
	peer.sendMessage(message)
	response, err := peer.waitForMessage(xdr.MessageTypeScpQuorumset)
	if err != nil {
		panic(err)
	}
	qset, ok := response.GetQSet()
	if ok {
		return qset
	}
	fmt.Printf("Response is not qset: %v\n", response)
	panic("omg..")
}

// GetScpState gets the scp state
func (peer *Peer) GetScpState() *xdr.StellarMessage {
	command := 0
	message, err := xdr.NewStellarMessage(xdr.MessageTypeGetScpState, command)
	if err != nil {
		fmt.Println(err)
	}

	peer.sendMessage(message)
	r, _ := peer.waitForMessage(xdr.MessageTypeScpMessage)
	return r
}
