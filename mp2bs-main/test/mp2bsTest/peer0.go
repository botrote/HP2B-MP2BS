package main

import (
	"flag"
	"fmt"
	"mp2bs/mp2session"
)

const blockNumber = uint32(10)

func main() {

	fileName := flag.String("b", "block", "File name you want to send")
	blockNumber := flag.Int("n", 10, "Block number(any value available)")
	configFile := flag.String("c", "peer0.toml", "Config file name")
	peerId := flag.Int("i", 0, "Peer ID")
	testFlag := flag.Bool("t", false, "TEST mode")
	rate := flag.Float64("r", 100, "Throughput(unit: mbit)")
	time := flag.Float64("e", 10, "Running time(unit: sec)")
	flag.Parse()

	fmt.Printf("PEER-%d: Config file name=%s \n\n\n", *peerId, *configFile)

	// Create MP2 Session Manager
	mp2 := mp2session.CreateMp2SessionManager(*configFile, uint32(*peerId))

	// Accept connection from parent and connect to children if non-leaf node
	go mp2.Mp2Listen()

	if !*testFlag {
		// Send block
		// We assume that the requested block is "block" in this test application
		fmt.Printf("PEER-%d: File name=%s_%d \n", *peerId, *fileName, *blockNumber)
		fmt.Printf("PEER-%d: Send a block(number=%d) \n", *peerId, *blockNumber)
		mp2.Mp2SendBlock(*fileName, uint32(*blockNumber))
	} else {
		// TP TEST (default rate: 100mbit, time: 10sec)
		// We calculate the size of the block so that it can run for as long as the entered time
		fmt.Printf("PEER-%d: Throughput test\n", *peerId)
		mp2.Mp2Test(*rate, *time) // Mp2Test(unit: mbit, unit: sec)
	}

	mp2.Mp2Close()

	fmt.Printf("PEER-%d: FINISH!!! \n\n", *peerId)
}
