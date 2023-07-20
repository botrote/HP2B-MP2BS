package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

const size = 65536

func main() {

	myAddr := flag.String("m", "127.0.0.1:5000", "My address")
	flag.Parse()

	listener, err := net.Listen("tcp", *myAddr)
	if err != nil {
		panic(err)
	}

	stream, err := listener.Accept()

	s, _ := os.Create("throughput_tcp")
	defer s.Close()

	total := 0
	count := 0
	throughput := 0.0

	recvBuf := make([]byte, size)

	startTime := time.Now()

	for {
		n, err := stream.Read(recvBuf)
		if err != nil {
			if err == io.EOF {
				stream.Close()
				break
			}
		}

		total += n
		count++

		elapsedTime := time.Since(startTime)
		throughput = (float64(total) * 8.0) / float64(elapsedTime.Seconds()) / (1000 * 1000)

		if count%1000 == 0 {
			fmt.Printf("Seconds=%f, Throughput=%f\n", elapsedTime.Seconds(), throughput)
			log := fmt.Sprintf("Seconds=%f, Throughput=%f\n", elapsedTime.Seconds(), throughput)
			s.Write([]byte(log))
		}
	}

	fmt.Printf("Complete!! (%d bytes)\n", total)
}
