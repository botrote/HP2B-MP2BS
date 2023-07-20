package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"time"

	quic "github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"github.com/quic-go/quic-go/qlog"
)

type bufferedWriterCloser struct {
	*bufio.Writer
	io.Closer
}

const size = 65536

func main() {

	myAddr := flag.String("m", "127.0.0.1:5000", "My address")
	logFlag := flag.Bool("l", false, "Quic log output")
	flag.Parse()

	tlsConfig := generateTLSConfig()

	quicConf := &quic.Config{}

	if *logFlag {
		quicConf.Tracer = qlog.NewTracer(func(p logging.Perspective, connectionID []byte) io.WriteCloser {
			filename := fmt.Sprintf("receiver_%x.qlog", connectionID)
			f, err := os.Create(filename)
			if err != nil {
				panic(err)
			}
			return &bufferedWriterCloser{
				Writer: bufio.NewWriter(f),
				Closer: f,
			}
		})
	}

	listener, err := quic.ListenAddr(*myAddr, tlsConfig, quicConf)
	if err != nil {
		panic(err)
	}

	quicSession, _ := listener.Accept(context.Background())

	quicStream, _ := quicSession.AcceptStream(context.Background())

	s, _ := os.Create("throughput_quic")
	defer s.Close()

	total := 0
	count := 0
	throughput := 0.0

	recvBuf := make([]byte, size)

	startTime := time.Now()

	for {
		n, err := quicStream.Read(recvBuf)
		if err != nil {
			if strings.Contains(string(err.Error()), "CLOSE") {
				quicStream.Close()
				quicSession.CloseWithError(0, "CLOSE")
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

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"mp2bs"},
		// MaxVersion:   tls.VersionTLS13,
		// MinVersion:   tls.VersionTLS13,
	}
}
