package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"
	"github.com/tinco/stellar-core-go/nodeInfo"
	"github.com/tinco/stellar-core-go/peer"
)

// TODO we somehow want to know about the NodeID.. maybe we filter based on node
// id and then just log whenever the node id is not our node id?

// map of quorum set hashes to their owners
var quorumSetHashes map[xdr.Hash]map[xdr.NodeId]bool
var p *peer.Peer

func main() {
	log.Println("Stellar Go Debug Client")

	quorumSetHashes = make(map[xdr.Hash]map[xdr.NodeId]bool)

	nodeInfo := nodeInfo.SetupCrypto()

	peerAddress := os.Args[1]

	var err error
	p, err = peer.Connect(&nodeInfo, peerAddress)
	if err != nil {
		log.Fatal("Couldn't connect to ", peerAddress)
	}

	quorumSetMessagesChan := make(chan *xdr.StellarMessage, 1)

	p.OnMessage = func(message *xdr.StellarMessage) {
		switch message.Type {
		case xdr.MessageTypeScpMessage:
			handleSCPMessage(message)
		case xdr.MessageTypeScpQuorumset:
			quorumSetMessagesChan <- message
		case xdr.MessageTypeErrorMsg:
			err := message.MustError()
			log.Printf("Got error message: %s\n", err.Msg)
		case xdr.MessageTypeDontHave:
			dontHave := message.MustDontHave()
			log.Printf("Received donthave: %v, %v\n", dontHave.ReqHash, dontHave.Type)
		default:
			//log.Printf("Unsolicited message: %v\n", message.Type)
		}
	}

	p.Start()

	time.Sleep(30 * time.Second) // sometimes updates are slow to come in?

	for hash, owners := range quorumSetHashes {
		p.GetScpQuorumset(hash)
		select {
		case msg := <-quorumSetMessagesChan:
			for owner := range owners {
				qs := handleScpQuorumSet(msg, owner)
				fmt.Println(qs)
			}
		case <-time.After(3 * time.Second):
			log.Fatalf("Timed out waiting for quorum set message")
		}
	}
}

func gotNewHash(hash xdr.Hash) {
	log.Printf("Requesting qset: %s", hex.EncodeToString(hash[:]))
	p.GetScpQuorumset(hash)
}

func handleScpQuorumSet(message *xdr.StellarMessage, owner xdr.NodeId) string {
	qs := message.MustQSet()
	log.Printf("Received qset")
	prepared := prepQuorumSet(qs)
	pkey := owner.MustEd25519()
	prepared["owner"], _ = strkey.Encode(strkey.VersionByteAccountID, pkey[:])
	jsDump, err := json.Marshal(prepared)
	if err != nil {
		log.Fatal("Could not dump json of quorumset")
	}
	return string(jsDump)
}

func prepQuorumSet(qs xdr.ScpQuorumSet) map[string]interface{} {
	validators := qs.Validators
	innerSets := qs.InnerSets
	threshold := qs.Threshold

	data := make(map[string]interface{})
	vals := make([]string, len(validators))

	for i, v := range validators {
		pk := v.MustEd25519()
		pks, _ := strkey.Encode(strkey.VersionByteAccountID, pk[:])
		vals[i] = pks
	}

	data["threshold"] = threshold
	data["validators"] = vals

	ins := make([]interface{}, len(innerSets))
	for i, v := range innerSets {
		ins[i] = prepQuorumSet(v)
	}

	data["inner_sets"] = ins

	return data
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
		quorumSetHashes[qs] = make(map[xdr.NodeId]bool)
	}
	quorumSetHashes[qs][envelope.Statement.NodeId] = true
}

func handleSCPMessage(message *xdr.StellarMessage) {
	envelope, ok := message.GetEnvelope()
	if ok {
		trackQuorumSetHashes(envelope)
	} else {
		fmt.Printf("{ \"error\": \"Got some unexpected SCP message type: %v\"}\n", message)
	}
}
