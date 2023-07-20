package main

import (
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"time"
)

const size = 65536

func main() {
	childAddr := flag.String("c", "127.0.0.1:5000", "Child address")
	t := flag.Float64("t", 10.0, "Total time")
	flag.Parse()

	conf := &tls.Config{
		InsecureSkipVerify: true,
		MaxVersion:         tls.VersionTLS13,
		MinVersion:         tls.VersionTLS10,
	}

	startTime := time.Now()

	// Connect to Master listener
	stream, err := tls.Dial("tcp", *childAddr, conf)
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
