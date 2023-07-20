package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

const OmtreeIP string = "127.0.0.1"
const OmtreePortNum string = "12345"

const NodeTotal uint = 80
const Ndim uint = 6

func main() {
	conn, err := net.Dial("tcp", OmtreeIP+":"+OmtreePortNum)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Conn fin!")

	msg := make([]byte, 4096)
	opt := 0
	binary.BigEndian.PutUint32(msg[0:4], uint32(NodeTotal))
	binary.BigEndian.PutUint32(msg[4:8], uint32(Ndim))
	binary.BigEndian.PutUint32(msg[8:12], uint32(opt))

	conn.Write(msg)
	fmt.Println("sendCoordInfo to omtree fin!")
}
