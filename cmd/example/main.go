package main

import (
	"flag"
	"github.com/ebuchman/debora"
)

var (
	deboraDev = flag.Bool("debora-dev", false, "allow debora management (dev side)")
	deboraM   = flag.Bool("debora", false, "allow remote developer to manage the process")
)

var (
	key  = []byte("abcdefghijklmnopqrst1234567890alabcdefghijklmnopqrst1234567890al") // developer's 64 byte key
	host = "0.0.0.0:8009"                                                             // developer's ip and port
)

func main() {
	flag.Parse()

	if *deboraM {
		debora.Add(key)
	}

	// connect to bootstrap node and run the protocol
	go runProtocol()

	// hook for developer to broadcast upgrade message to all peers
	if *deboraDev {
		debora.Master("example", broadcast)
	}

}

func runProtocol() {
	// connect to bootstrap node
	// select loop over protocol
	// case MsgDeboraTy: debora.Call()

}

func broadcast() {
	// broadcast an empty MsgDeboraTy message to all peers
}
