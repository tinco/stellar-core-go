package main

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/stellar/go/xdr"
	"github.com/tinco/stellar-core-go/nodeInfo"
	"github.com/tinco/stellar-core-go/peer"
)

var quorumSetHashes map[xdr.Hash]bool

func main() {
	fmt.Println("Stellar Go Debug Client\n ")

	quorumSetHashes = make(map[xdr.Hash]bool)

	nodeInfo := nodeInfo.SetupCrypto()
	// "stellar0.keybase.io:11625")
	peer, err := peer.Connect(&nodeInfo, "localhost:11625")
	if err != nil {
		panic("Couldn't connect")
	}

	peer.OnMessage = func(message xdr.StellarMessage) {
		switch message.Type {
		case xdr.MessageTypePeers:
			handlePeers(message)
		case xdr.MessageTypeScpMessage:
			handleSCPMessage(message)
		default:
			fmt.Printf("Unsolicited message: %v\n", message.Type)
		}
	}

	peer.Start()

	peer.GetPeerAddresses()

	for {
		time.Sleep(100 * time.Millisecond)
	}
}

func handlePeers(message xdr.StellarMessage) {
	peers := message.MustPeers()
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
	fmt.Printf("Addresses: %v\n", peerAddresses)
}

func gotNewHash(hash xdr.Hash) {
	fmt.Printf("Got new quorumSetHash: %v\n", hash)
}

func trackQuorumSetHashes(envelope xdr.ScpEnvelope) {
	var qs xdr.Hash
	switch envelope.Statement.Pledges.Type {
	case xdr.ScpStatementTypeScpStNominate:
		qs = envelope.Statement.Pledges.MustNominate().QuorumSetHash
	case xdr.ScpStatementTypeScpStExternalize:
		qs = envelope.Statement.Pledges.MustExternalize().CommitQuorumSetHash
	case xdr.ScpStatementTypeScpStPrepare:
		qs = envelope.Statement.Pledges.MustPrepare().QuorumSetHash
	case xdr.ScpStatementTypeScpStConfirm:
		qs = envelope.Statement.Pledges.MustConfirm().QuorumSetHash
	}
	_, exists := quorumSetHashes[qs]
	if !exists {
		quorumSetHashes[qs] = true
		gotNewHash(qs)
	}
}

func handleSCPMessage(message xdr.StellarMessage) {
	envelope, ok := message.GetEnvelope()
	if ok {
		trackQuorumSetHashes(envelope)
	} else {
		fmt.Printf("Got some unexpected SCP message type: %v", message)
	}
}
