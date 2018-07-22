package main

import (
	"fmt"
	"time"

	"github.com/stellar/go/xdr"
	"github.com/tinco/stellar-core-go/nodeInfo"
	"github.com/tinco/stellar-core-go/peer"
)

func main() {
	fmt.Println("Stellar Go Debug Client\n ")

	nodeInfo := nodeInfo.SetupCrypto()
	// "stellar0.keybase.io:11625")
	peer, err := peer.Connect(&nodeInfo, "localhost:11625")
	if err != nil {
		panic("Couldn't connect")
	}

	hashes := make([]xdr.Hash, 50)

	peer.OnQuorumSetHash = func(hash xdr.Hash) {
		qs := peer.GetScpQuorumset(hash)
		fmt.Printf("Got qset: %v\n", qs)
		hashes = append(hashes, hash)
	}

	peer.Start()

	peerAddresses := peer.GetPeerAddresses()
	fmt.Printf("Addresses: %v\n\n", peerAddresses)

	time.Sleep(2000 * time.Millisecond)

	/*
		scpState := peer.GetScpState()
		fmt.Printf("ScpState: %v\n\n", scpState)
	*/
	for {
		time.Sleep(100 * time.Millisecond)
	}
}
