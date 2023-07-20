package main

import (
	"flag"
	"fmt"
	"mp2bs/mp2session"
	"os"
	"time"
)

const size = 32768

func main() {
	var pb PB
	var blockNumber uint32
	var blockSize uint32

	configFile := flag.String("c", "peers.toml", "Config file name")
	peerId := flag.Int("i", 2, "Peer ID")
	flag.Parse()

	fmt.Printf("PEER-%d: Config file name=%s \n\n\n", *peerId, *configFile)

	// Create MP2 Session Manager
	mp2 := mp2session.CreateMp2SessionManager(*configFile, uint32(*peerId))

	// Accept connection from parent and connect to children if non-leaf node
	go mp2.Mp2Listen()

	// TODO: (Temporary) Wait until block info is received
	// go-routine?
	for {
		blockNumber, blockSize = mp2.Mp2GetBlockInfo()
		if blockNumber >= 0 && blockSize > 0 {
			break
		}
	}

	// File open to write
	filename := fmt.Sprintf("peer_%d_block_%d", *peerId, blockNumber)
	fo, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	modeStatus := mp2.Mp2GetPeerStatus()
	pb.Show(0, filename, 0, 0, modeStatus)

	pb.SetOption(0, int64(blockSize))

	s, _ := os.Create(fmt.Sprintf("receiver_throughput_PEER%d\n", *peerId))
	defer s.Close()

	startTime := time.Now()

	total := 0
	throughput := 0.0
	count := 0

	for {
		// Read block data from MP2 Session
		buf := make([]byte, size)
		n, err := mp2.Mp2ReadBlock(buf, blockNumber)
		if err != nil {
			panic(err)
		}

		if n <= 0 {
			break
		}

		total = total + n
		count++

		elapsedTime := time.Since(startTime)
		throughput = (float64(total) * 8.0) / float64(elapsedTime.Seconds()) / (1000 * 1000)

		if count%1000 == 0 {
			log := fmt.Sprintf("Seconds=%f, Throughput=%f, ReceivedSize=%d\n", elapsedTime.Seconds(), throughput, total)
			s.Write([]byte(log))
		}

		// Write block data into file
		_, err = fo.Write(buf[:n])
		if err != nil {
			fmt.Printf("PEER-%d: Write error\n", *peerId)
			panic(err)
		}

		modeStatus = mp2.Mp2GetPeerStatus()
		pb.Show(int64(total), filename, float64(elapsedTime.Seconds()), throughput, modeStatus)
	}

	mp2.Mp2Close()

	fmt.Printf("PEER-%d: FINISH!!! \n\n", *peerId)

	pb.Finish()
}

// Progress Bar
type PB struct {
	percent   int64
	current   int64
	total     int64
	rate      string
	character string
}

func (pb *PB) SetOption(start, total int64) {
	pb.current = start
	pb.total = total
	pb.character = "#"
	pb.percent = pb.getPercent()

	for i := 0; i < int(pb.percent); i += 2 {
		pb.rate += pb.character
	}
}

func (pb *PB) getPercent() int64 {
	return int64((float32(pb.current) / float32(pb.total)) * 100)
}

func (pb *PB) Show(current int64, fileName string, time float64, throughput float64, modeStatus string) {
	pb.current = current
	last := pb.percent
	pb.percent = pb.getPercent()

	// If there is a difference between the last percent and the current percent
	if pb.percent != last && pb.percent%4 == 0 {
		for i := 0; i < int(pb.percent-last); i += 2 {
			pb.rate += pb.character
		}
	}

	// Loading information
	fmt.Printf("\033[F\033[F\033[F[Status] %s \n", modeStatus)
	fmt.Printf("%s [%-25s] %3d%% %8d/%d  %.2fMbps in %.3fs \n", fileName, pb.rate, pb.percent, pb.current, pb.total, throughput, time)
}

func (pb *PB) Finish() {
	fmt.Println()
}
