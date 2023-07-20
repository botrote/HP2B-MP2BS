// self-maintenance
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

const pullInterval int = 2
const recvByteSize uint = 8

const verIpAddr string = "172.17.0.2"
const verPortNum string = "9999"

var meshNeedUpdate bool = false

const (
	typePullReq uint16 = iota
	typePullRes
	typeNoMissingBlk
	typeGetMissingBlk
)

const (
	nodeTypeClusterLeader uint8 = iota
	nodeTypeMesh
	nodeTypeNormal
)

const (
	resInFailure uint8 = iota
	resOutFailure
)

type MsgPullReq struct {
	msgType   uint8
	blkLength uint64
}

type meshNeighbor struct {
	ipAddr   string
	nodeType uint8
}

type Node struct {
	nodeIdx    uint
	myIp       string
	portNum    string // TODO: for Local Test
	children   []string
	clusterNum uint
	isMesh     bool
	neighbors  []meshNeighbor
	// meshNeighbors []string
	curBlkLength uint64
}

var myNode Node

func initNode(args []string) {
	myNode.nodeIdx = 1
	myNode.myIp = args[1]

	// *********** hyunmin 수정 *************//
	//myNode.portNum = args[2]
	udp_port_int, _ := strconv.Atoi(args[2]) // 200을 더해서 u
	udp_port_int = udp_port_int + 200        // 정수를 다시 문자열로 변환
	myNode.portNum = strconv.Itoa(udp_port_int)
	myNode.isMesh = false //isMesh != 0 // defualt 는 mesh 노드 아님

	// *********** hyunmin 수정 *************//
	myNode.curBlkLength, _ = strconv.ParseUint(myNode.portNum, 10, 64)
}

func addMeshNeighbor(ipLists []string, typeLists []uint8) {
	var i int = 0
	fmt.Println("addMeshNeightbor(): ", len(ipLists))
	for _, ip := range ipLists {
		nb := meshNeighbor{
			ipAddr:   ip,
			nodeType: typeLists[i],
		}
		myNode.neighbors = append(myNode.neighbors, nb)
		fmt.Println(i, "["+peer_name+"] add Mesh Neighbor:", ip, "type:", nb.nodeType)
		i += 1
	}
}

func sendResToVer(resMsgtoVer uint8) {
	fmt.Println("sendResToVer:", resMsgtoVer)
	switch resMsgtoVer {
	case resInFailure:
		fmt.Println("In failure!!")
	case resOutFailure:
		fmt.Println("Out failure!!")
	}

	remoteAddr, err := net.ResolveUDPAddr("udp4", verIpAddr+":"+verPortNum)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	msg := make([]byte, 6) // Assuming uint16 takes 2 bytes (In/Out), uint32 takes 4 bytes (clusterNum when inFailure)

	// Convert the integer and uint64 to byte slices
	binary.BigEndian.PutUint16(msg[0:2], uint16(resMsgtoVer))
	if resMsgtoVer == resInFailure {
		binary.BigEndian.PutUint32(msg[2:6], uint32(myNode.clusterNum))
	}

	fmt.Println("self maintenance report to verification node")
	_, err = conn.Write(msg)
	if err != nil {
		log.Fatal(err)
	}
}

func sendPullRequest() int {
	var maxBlkLength uint64 = 0
	var nodeReqMissingBlk meshNeighbor
	idx := 0
	for _, nb := range myNode.neighbors {
		remoteAddr, err := net.ResolveUDPAddr("udp", nb.ipAddr)
		if err != nil {
			log.Fatal(err)
		}

		conn, err := net.DialUDP("udp", nil, remoteAddr)
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		// Create a byte slice to hold the message data
		msg := make([]byte, 10) // Assuming int16 takes 4 bytes, and uint64 takes 8 bytes

		// Convert the integer and uint64 to byte slices
		binary.BigEndian.PutUint16(msg[0:2], uint16(typePullReq))
		binary.BigEndian.PutUint64(msg[2:10], myNode.curBlkLength)

		fmt.Println("Mesh Pull Request to " + nb.ipAddr)
		_, err = conn.Write(msg)
		if err != nil {
			log.Fatal(err)
		}

		buf := make([]byte, 10)
		n, _, err := conn.ReadFromUDP(buf)
		msgtype := uint16(binary.BigEndian.Uint16(buf[0:2]))
		recvBlkLength := uint64(binary.BigEndian.Uint64(buf[2:10]))
		if n != 10 {
			panic("Received message has incorrect length")
		}

		recvBlkLength += 1

		switch msgtype {
		case typePullRes:
			if recvBlkLength > myNode.curBlkLength {
				// 누구한테 요청할지 기록
				if maxBlkLength < recvBlkLength {
					maxBlkLength = recvBlkLength
					nodeReqMissingBlk = nb
				}
			} else {
				fmt.Println("No missing block:", nb.ipAddr)
			}
		}
		idx += 1
	}
	// missing block이 있었을 경우, missing block을 요청하고, 검증노드에게 report
	var resPullReq uint8 = 0
	if maxBlkLength > 0 {
		fmt.Print(maxBlkLength-myNode.curBlkLength, " missing blocks from ")
		switch nodeReqMissingBlk.nodeType {
		case nodeTypeClusterLeader:
			fmt.Println("the cluster leader node:", nodeReqMissingBlk.ipAddr)
			resPullReq = resInFailure
		case nodeTypeMesh:
			fmt.Println("other mesh node:", nodeReqMissingBlk.ipAddr)
			resPullReq = resOutFailure
		case nodeTypeNormal:
			fmt.Println("normal node:", nodeReqMissingBlk.ipAddr)
			resPullReq = resInFailure
		}
		for blkNum := myNode.curBlkLength + 1; blkNum <= uint64(maxBlkLength); blkNum++ {
			fmt.Println("Get missing block: ", blkNum)
			requestAndReceiveMissingBlock(nodeReqMissingBlk.ipAddr, blkNum)
		}
		sendResToVer(resPullReq)
		// TODO: sendMissingBlktoAll() using Gossip / unicast(tcp)
	}

	return 0
}

func sendMissingBlktoAll() {
	panic("sendMissingBlktoAll not implemented")
}

func handleUDPConnection(conn *net.UDPConn) {
	buffer := make([]byte, 10) // Assuming int16 takes 2 bytes, and uint64 takes 8 bytes

	// udp connection으로 부터 값을 읽어들인다.
	n, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		log.Fatal(err)
	}

	if n != 10 {
		panic("Received message has incorrect length")
	}
	msgtype := uint16(binary.BigEndian.Uint16(buffer[0:2]))
	recvBlkLength := binary.BigEndian.Uint64(buffer[2:10])

	// 리턴 값은 전달 받은 클라이언트 서버의 address, msg
	fmt.Println("Received from UDP client (msgType:", msgtype, "recvBlkLength:", recvBlkLength, ") curBlkLength:", myNode.curBlkLength)

	msg := make([]byte, 10) // Assuming int32 takes 2 bytes, and uint64 takes 8 bytes

	binary.BigEndian.PutUint16(msg[0:2], typePullRes)
	binary.BigEndian.PutUint64(msg[2:10], myNode.curBlkLength)

	_, err = conn.WriteToUDP(msg, addr)
	if err != nil {
		log.Fatal(err)
	}
}

func MeshHandler() {
	udpAddr, err := net.ResolveUDPAddr("udp4", myNode.myIp+":"+myNode.portNum)
	if err != nil {
		log.Fatal(err)
	}

	// udp endpoint를 파라미터로 넘기면 udp connection을 리턴한다.
	listen, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal(err)
	}

	// listen하고 있는 상태
	fmt.Println("UDP server up and listening on port", myNode.portNum)
	defer listen.Close()

	for {
		handleUDPConnection(listen)
	}
}

func MeshRun() {
	time.Sleep(time.Duration(2) * time.Second) // wait until other nodes start
	fmt.Println("MeshRun!")
	for {
		if meshNeedUpdate {
			break
		}
		time.Sleep(time.Duration(pullInterval) * time.Second)

		fmt.Println("send Pull Request!")
		sendPullRequest()
	}
}
