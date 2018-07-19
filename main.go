package main

import (
	"fmt"

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

	peerAddresses := peer.GetPeerAddresses()
	fmt.Printf("Addresses: %v\n\n", peerAddresses)

	/*
		scpState := peer.GetScpState()
		fmt.Printf("ScpState: %v\n\n", scpState)

		quorumset := peer.GetScpQuorumset()
		fmt.Printf("Quorumset: %v\n\n", quorumset)

		txset := peer.GetTxSet()
		fmt.Printf("TxSet: %v\n\n", txset)*/

	quorumSets := make(map[xdr.Hash]bool)

	scpChan, _ := peer.WaitForMessages(xdr.MessageTypeScpMessage)
	func() {
		for {
			scpMessage := (<-scpChan)
			envelope, ok := scpMessage.GetEnvelope()
			if ok {
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
				_, exists := quorumSets[qs]
				if !exists {
					quorumSets[qs] = true
					fmt.Printf("Got qs: %v\n", qs)
				}
			} else {
				fmt.Printf("Got something else: %v", scpMessage)
			}
		}
	}()
}
