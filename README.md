# Stellar-Core in Go

This project is an implementation of the Stellar-Core Protocol (SCP) in Go. You can use its libraries to setup a connection to the Stellar-Core network, get information from nodes and even broadcast transactions.

This implementation leans heavily on the `xdr_generated.go` from the `stellar/go` library. The most significant contribution is `handshake.go` which performs the authentication handshake.

Besides being a library to connect to the Stellar-Core network it also has some commands that interact with the network.

***THIS PROJECT IS A WORK IN PROGRESS AND ITS API IS HIGHLY UNSTABLE***

## Examples

A simple connection to the Stellar-Core network can be established like so:

```
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/stellar/go/xdr"
	"github.com/tinco/stellar-core-go/nodeInfo"
	"github.com/tinco/stellar-core-go/peer"
)

func main() {
	nodeInfo := nodeInfo.SetupCrypto()

	peerAddress := os.Args[1]

	p, err := peer.Connect(&nodeInfo, peerAddress)
	if err != nil {
		log.Fatal("Couldn't connect to ", peerAddress)
	}

	p.OnMessage = func(message xdr.StellarMessage) {
		switch message.Type {
		case xdr.MessageTypeErrorMsg:
			err := message.MustError()
			log.Fatal("Got error message: ", err.Msg)
		default:
			fmt.Printf("Received message: %v\n", message.Type)
		}
	}

	p.Start()
	time.Sleep(3000 * time.Millisecond)
	log.Fatal("Stopped after listening for 3 seconds")
}
```

For more examples look in the `cmd/` directory.

## Commands

This project contains some commands used for crawling the Stellar-Core network and obtaining information from nodes. Run `make` to compile them.

### `bin/peers`

Run `./bin/peers <somenode>:<itsport>` to get a list of peers this node connects to.

### `bin/quorumsets`

Run `./bin/quorumsets <somenode>:<itsport>` to get a stream of json objects with the quorumsets this node receives SCP messages from.

### `get_all_peers.rb`

Run `./get_all_peers.rb` to recursively discover all of the peers on the network. At the time of writing, running this command connects to about 8000 nodes, succesfully connecting to about 80.

## Motivation

This project is being developed as part of an effort to establish insight into the Stellar-Core network as suggested by the 7th Stellar Build challenge's project idea of "Building a better Quorum Explorer". 

If you got any questions, ideas, suggestions or large piles of Lumens to send, please contact me at https://keybase.io/tinco
