package main

import (
	"flag"
	"fmt"
	"mp2bs/mp2session"
	"time"
)

func main() {
	configFile := flag.String("c", "anchor.toml", "Config file name")
	topAddr := flag.String("t", "127.0.0.1:5000", "Top node")
	flag.Parse()

	fmt.Printf("ANCHOR_PEER: Config file name=%s \n\n\n", *configFile)

	// Create MP2 Session Manager
	mp2 := mp2session.CreateMp2SessionManager(*configFile, 100)

	// Accept connection from parent and connect to children if non-leaf node
	go mp2.Mp2Listen()

	// Connect to top node
	mp2.Mp2Connect(*topAddr)

	time.Sleep(1 * time.Second)

	// Send block
	fmt.Printf("ANCHOR_PEER: Send Node info packet\n")
	mp2.Mp2SendNodeInfoForAnchor()

	// Do not use MP2Close() because of hardcoding
	time.Sleep(1 * time.Second)

	fmt.Printf("ANCHOR_PEER: FINISH!!! \n\n")
}
