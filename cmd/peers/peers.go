package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/stellar/go/xdr"
	"github.com/tinco/stellar-core-go/nodeInfo"
	"github.com/tinco/stellar-core-go/peer"
)

var p *peer.Peer

func main() {
	fmt.Println("Getting peers..\n ")

	nodeInfo := nodeInfo.SetupCrypto()
	// "stellar0.keybase.io:11625")

	var err error
	p, err = peer.Connect(&nodeInfo, "localhost:11625")
	if err != nil {
		panic("Couldn't connect")
	}

	p.OnMessage = func(message xdr.StellarMessage) {
		switch message.Type {
		case xdr.MessageTypePeers:
			handlePeers(message)
			os.Exit(0)
		default:
			// fmt.Printf("Unsolicited message: %v\n", message.Type)
		}
	}

	p.Start()
	p.GetPeerAddresses()

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
	fmt.Printf("Peer addresses: %v\n", peerAddresses)
}
