package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/stellar/go/strkey"
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
		fmt.Printf("{ \"error\": \"%s\"}", err.Error())
		return
	}

	p.OnMessage = func(message *xdr.StellarMessage) {
		switch message.Type {
		case xdr.MessageTypePeers:
			handleConnectionStart(message)
		case xdr.MessageTypeErrorMsg:
			err := message.MustError()
			fmt.Printf("{ \"error\": \"%s\"}", err.Msg)
		default:
			// fmt.Printf("Unsolicited message: %v\n", message.Type)
		}
	}

	p.Start()
	time.Sleep(3 * time.Second)
	fmt.Printf("{ \"ok\": \"Stopped listening after 3 seconds\"}")
}

func handleConnectionStart(message *xdr.StellarMessage) {
	peers := getPeers(message)
	peerInfo := make(map[string]interface{})
	peerInfo["peers"] = peers
	peerInfo["info"] = prepInfo(p.PeerInfo)
	serialized, _ := json.Marshal(peerInfo)
	fmt.Println(string(serialized))
}

func prepInfo(hello *xdr.Hello) interface{} {
	info := make(map[string]interface{})
	info["network_id"] = hex.EncodeToString(hello.NetworkId[:])
	info["ledger_version"] = hello.LedgerVersion
	info["peer_id"] = strkey.MustEncode(strkey.VersionByteAccountID, hello.PeerId.Ed25519[:])
	info["overlay_version"] = hello.OverlayVersion
	info["overlay_min_version"] = hello.OverlayMinVersion
	info["version_string"] = hello.VersionStr

	return info
}

func getPeers(message *xdr.StellarMessage) []string {
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
	return peerAddresses
}
