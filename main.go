package main

import (
	"fmt"

	"github.com/tinco/stellar-core-go/handshake"
	"github.com/tinco/stellar-core-go/nodeInfo"
	"github.com/tinco/stellar-core-go/peer"
)

func main() {
	fmt.Println("Stellar Go Debug Client")

	nodeInfo := nodeInfo.SetupCrypto()
	// "stellar0.keybase.io:11625")
	peer, err := peer.Connect("localhost:11625")
	if err != nil {
		panic("Couldn't connect")
	}

	handshake.StartAuthentication(&nodeInfo, peer)
}
