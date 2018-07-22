package main

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
	"time"

	"github.com/stellar/go/xdr"
	"github.com/tinco/stellar-core-go/nodeInfo"
	"github.com/tinco/stellar-core-go/peer"
)

var quorumSetHashes map[xdr.Hash]string
var p *peer.Peer

func main() {
	fmt.Println("Stellar Go Debug Client\n ")

	quorumSetHashes = make(map[xdr.Hash]string)

	nodeInfo := nodeInfo.SetupCrypto()
	// "stellar0.keybase.io:11625")

	var err error
	p, err = peer.Connect(&nodeInfo, "localhost:11625")
	if err != nil {
		panic("Couldn't connect")
	}

	p.OnMessage = func(message xdr.StellarMessage) {
		switch message.Type {
		case xdr.MessageTypeScpMessage:
			handleSCPMessage(message)
		case xdr.MessageTypeScpQuorumset:
			handleScpQuorumSet(message)
		default:
			// fmt.Printf("Unsolicited message: %v\n", message.Type)
		}
	}

	p.Start()

	for {
		time.Sleep(100 * time.Millisecond)
	}
}

func gotNewHash(hash xdr.Hash) {
	fmt.Printf("Got new quorumSetHash: %v\n", quorumSetHashes[hash])
	p.GetScpQuorumset(hash)
}

func handleScpQuorumSet(message xdr.StellarMessage) {
	qs := message.MustQSet()
	jsDump, err := json.Marshal(qs)
	if err != nil {
		fmt.Printf("Could not dump json of quorumset")
	}
	fmt.Printf("QuorumSet JSON: %s\n", jsDump)
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
		encoded := base32.StdEncoding.EncodeToString(qs[:])
		quorumSetHashes[qs] = encoded
		gotNewHash(qs)
	}
}

func handleSCPMessage(message xdr.StellarMessage) {
	envelope, ok := message.GetEnvelope()
	if ok {
		trackQuorumSetHashes(envelope)
	} else {
		fmt.Printf("{ \"error\": \"Got some unexpected SCP message type: %v\"}\n", message)
	}
}
