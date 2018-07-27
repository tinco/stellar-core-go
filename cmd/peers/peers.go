package main

import (
	"encoding/json"
	"fmt"
	"log"
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
	nodeInfo := nodeInfo.SetupCrypto()

	peerAddress := os.Args[1]

	var err error
	p, err = peer.Connect(&nodeInfo, peerAddress)
	if err != nil {
		log.Fatal("Couldn't connect to ", peerAddress)
	}

	p.OnMessage = func(message *xdr.StellarMessage) {
		switch message.Type {
		case xdr.MessageTypePeers:
			handlePeers(message)
			os.Exit(0)
		case xdr.MessageTypeErrorMsg:
			err := message.MustError()
			log.Fatal("Got error message: ", err.Msg)
		default:
			// fmt.Printf("Unsolicited message: %v\n", message.Type)
		}
	}

	p.Start()
	time.Sleep(3000 * time.Millisecond)
	log.Fatal("Peer did not respond within 3 seconds")
}

func handlePeers(message *xdr.StellarMessage) {
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
	addressesJSON, err := json.MarshalIndent(peerAddresses, "", "    ")
	if err != nil {
		log.Fatal("Could not marshal peer addresses into json.")
	}
	fmt.Println(string(addressesJSON))
}
