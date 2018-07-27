package main

import (
	"encoding/base32"
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

var quorumSetHashes map[xdr.Hash]string
var p *peer.Peer

func main() {
	fmt.Println("Stellar Go Debug Client\n ")

	quorumSetHashes = make(map[xdr.Hash]string)

	nodeInfo := nodeInfo.SetupCrypto()

	peerAddress := os.Args[1]

	var err error
	p, err = peer.Connect(&nodeInfo, peerAddress)
	if err != nil {
		log.Fatal("Couldn't connect to ", peerAddress)
	}

	p.OnMessage = func(message *xdr.StellarMessage) {
		switch message.Type {
		case xdr.MessageTypeScpMessage:
			handleSCPMessage(message)
		case xdr.MessageTypeScpQuorumset:
			handleScpQuorumSet(message)
		case xdr.MessageTypeErrorMsg:
			err := message.MustError()
			fmt.Printf("Got error message: %s\n", err.Msg)
		case xdr.MessageTypeDontHave:
			dontHave := message.MustDontHave()
			fmt.Printf("Received donthave: %v, %v", dontHave.ReqHash, dontHave.Type)
		default:
			//fmt.Printf("Unsolicited message: %v\n", message.Type)
		}
	}

	p.Start()

	for {
		time.Sleep(100 * time.Millisecond)
	}
}

func gotNewHash(hash xdr.Hash) {
	p.GetScpQuorumset(hash)
}

func handleScpQuorumSet(message *xdr.StellarMessage) {
	qs := message.MustQSet()
	prepared := prepQuorumSet(qs)
	jsDump, err := json.Marshal(prepared)
	if err != nil {
		fmt.Printf("Could not dump json of quorumset")
	}
	fmt.Println(string(jsDump))
}

func prepQuorumSet(qs xdr.ScpQuorumSet) interface{} {
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
	if envelope.Statement.Pledges.Type == xdr.ScpStatementTypeScpStExternalize {
		qs := envelope.Statement.Pledges.MustExternalize().CommitQuorumSetHash
		_, exists := quorumSetHashes[qs]
		if !exists {
			encoded := base32.StdEncoding.EncodeToString(qs[:])
			quorumSetHashes[qs] = encoded
			gotNewHash(qs)
		}
	}
}

func handleSCPMessage(message *xdr.StellarMessage) {
	envelope, ok := message.GetEnvelope()
	if ok {
		trackQuorumSetHashes(envelope)
	} else {
		fmt.Printf("{ \"error\": \"Got some unexpected SCP message type: %v\"}\n", message)
	}
}
