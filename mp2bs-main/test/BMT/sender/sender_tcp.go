package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"net"
	"time"
)

const size = 65536

func main() {
	childAddr := flag.String("c", "127.0.0.1:5000", "Child address")
	myAddr := flag.String("m", "127.0.0.1:4000", "My address")
	t := flag.Float64("t", 10.0, "Total time")
	flag.Parse()

	startTime := time.Now()

	// Bind remote IP
	rAddr, _ := net.ResolveTCPAddr("tcp", *childAddr)

	// Bind my IP
	lAddr, _ := net.ResolveTCPAddr("tcp", *myAddr)

	// Connect to Master listener
	stream, err := net.DialTCP("tcp", lAddr, rAddr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Connection complete(%f)\n", time.Since(startTime).Seconds())

	buf := make([]byte, size)
	total := 0
	n := 0

	if n, err = rand.Read(buf); err != nil {
		panic(err)
	}

	for {

		total = total + n

		stream.Write(buf[:n])

		elapsedTime := time.Since(startTime)

		if elapsedTime.Seconds() >= *t {
			stream.Close()
			break
		}

	}

	fmt.Printf("Complete!! (%d bytes)\n", total)
}
