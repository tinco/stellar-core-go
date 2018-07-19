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
	peers := peer.receiveMessage().MustPeers()
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
func (peer *Peer) GetTxSet() {
}

// AnnounceTransaction informs peer of a transaction
func (peer *Peer) AnnounceTransaction() {
}

// GetScpQuorumset gets scp quorumstate
func (peer *Peer) GetScpQuorumset() {
}

// GetScpState gets the scp state
func (peer *Peer) GetScpState() {
}
