package peer

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

func (peer *Peer) trackQuorumSetHashes(envelope xdr.ScpEnvelope) {
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
	_, exists := peer.quorumSetHashes[qs]
	if !exists {
		peer.quorumSetHashes[qs] = true
		peer.OnQuorumSetHash(qs)
	}
}

func (peer *Peer) listenForSCPMessages() {
	scpChan, _ := peer.WaitForMessages(xdr.MessageTypeScpMessage)
	go func() {
		for {
			scpMessage := (<-scpChan)
			envelope, ok := scpMessage.GetEnvelope()
			if ok {
				peer.trackQuorumSetHashes(envelope)
			} else {
				fmt.Printf("Got some unexpected SCP message type: %v", scpMessage)
			}
		}
	}()
}
