package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"mp2bs/mp2session"
)

/*
what to replace net.Conn.RemoteAddr() ?

*/


type ClientSession struct {
	mp2   mp2session.Mp2SessionManager
	active bool
}

var connections map[string]mp2session.Mp2SessionManager
var peer_name string
var blk_header_len = 64

func forwardBlockForAllChild(child_mp2s map[string]mp2session.Mp2SessionManager, buffer []byte) {
	for child_addr, child_con := range connections {
		fmt.Printf("["+peer_name+"] send from %s block to %s \n", peer_name, child_addr)
		go ForwardBlock(child_con, child_addr, buffer)
	}
}

func ForwardBlock(child mp2session.Mp2SessionManager, serverName string, buffer []byte) {

	_, err := child.Mp2SendBlock() //need to set param
	if err != nil {
		fmt.Println(err)
		return
	}
}

// 부모 노드 (client) 에 block 수신 및 자식 노드 (sever)로 블록 forwarding
func handleClient(session *ClientSession) {
	defer session.mp2.Mp2Close()

	buffer := make([]byte, 1024*100)

	for {
		n, err := session.mp2.Mp2ReadBlock() //need to set param
		if err != nil {
			fmt.Println("클라이언트 연결 종료:", session.conn.RemoteAddr()) //?
			break
		}

		data := string(buffer[:n])
		blk_num := uint64(binary.BigEndian.Uint64(buffer[0:blk_header_len]))

		if blk_num == 1 {
			myNode.curBlkLength = 1
			msg := buffer[:n]
			fmt.Printf("["+peer_name+"] 부모 노드에서 수신된 데이터 [%s]: %d %s\n", session.conn.RemoteAddr(), blk_num, data)

			if len(connections) != 0 {
				forwardBlockForAllChild(connections, msg)
			}

		} else if blk_num == myNode.curBlkLength+1 {
			myNode.curBlkLength = blk_num
			msg := buffer[:n]
			fmt.Printf("["+peer_name+"] 부모 노드에서 수신된 데이터 [%s]: %d %s\n", session.conn.RemoteAddr(), blk_num, data)

			if len(connections) != 0 {
				forwardBlockForAllChild(connections, msg)
			}

		} else {
			fmt.Printf("[" + peer_name + "] missing block 존재\n")

		}

	}
}

// 부모 노드 (client) 와 연결
func peerServer(ip, port string, wg *sync.WaitGroup, done chan bool) {
	defer wg.Done()

	clients := make(map[string]*ClientSession)
	var mutex sync.Mutex

	mp2 := mp2session.CreateMp2SessionManager(param?) //need set param
	go mp2.Mp2Listen()
	mp2.Mp2Connect(param?) //need set param
	for {
		blockNumber, blockSize = mp2.Mp2GetBlockInfo()
		if blockNumber >= 0 && blockSize > 0 {
			break
		}
	}

	/*
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("서버 시작 실패:", err)
		done <- true
		return
	}
	*/

	defer mp2.Mp2Close()

	mutex.Lock()

	fmt.Println("[" + peer_name + "] 부모 노드 " + conn.RemoteAddr().String() + " 와 연결")
	clients[conn.RemoteAddr().String()] = session

	mutex.Unlock()

}

func genBlock(blk_size int) {

	blk_idx := 1
	block := make([]byte, blk_size)

	for {
		// msg 생성
		msg := make([]byte, blk_header_len+blk_size)

		// 헤더 값 삽입
		binary.BigEndian.PutUint64(msg[0:blk_header_len], uint64(blk_idx))

		// 블록 데이터 복사
		copy(msg[blk_header_len:], block)

		if len(connections) != 0 {
			forwardBlockForAllChild(connections, msg)
		}
		blk_idx = blk_idx + 1

		time.Sleep(2 * time.Second)
	}

}

// GET THE IP OF THIS PEER NODE (JUST FOR DEBUGGING)
func get_local_ip() string {
	// this function is not that neccesary, but it can be useful when it comes to check the peer-side log.
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()
	ipAddress := conn.LocalAddr().(*net.UDPAddr)
	log.Printf("peer ip : %s\n\n", ipAddress)
	ipString := strings.Split(fmt.Sprintln(ipAddress), ":")
	return ipString[0]
}

func main() {
	ip := get_local_ip()
	port := "8000"
	args := []string{" ", ip, port} // os.Args
	if len(args) < 3 {
		fmt.Println("defulat IP:127.0.0.1, port: 8000")
		ip = "127.0.0.1"
		port = "8000"
	} else {
		ip = args[1]
		port = args[2]
		fmt.Println("set IP:" + ip + ", port: " + port)
	}

	// // 승규 코드 추가
	initNode(args)
	fmt.Println("IP:", myNode.myIp, "nodeIdx:", myNode.nodeIdx, "portNum:", myNode.portNum, "Mesh:", myNode.isMesh)

	peer_name = "peer_" + ip + ":" + port

	done := make(chan bool)
	var wg sync.WaitGroup

	wg.Add(1)

	// tree 구축 및 block forwarding 위한 server 실행
	go peerServer(ip, port, &wg, done)

	portInt, _ := strconv.Atoi(port)  // 100을 더함
	ctl_msg_port_int := portInt + 100 // 정수를 다시 문자열로 변환
	ctl_msg_port := strconv.Itoa(ctl_msg_port_int)

	go MeshHandler()

	// 검증 노드와 통신 및 cntl 수신을 위한 server 실행
	go peerVerServer(ip, ctl_msg_port, &wg, done)

	missing_block_trans_msg_port_int := portInt + 200 // 정수를 다시 문자열로 변환
	missing_block_trans_msg_port := strconv.Itoa(missing_block_trans_msg_port_int)

	go peerServerForMissingBlock(ip, missing_block_trans_msg_port, &wg, done)

	wg.Wait()

	//hp2b_client_closeConnections(connections)
	fmt.Println("모든 작업 완료")
}
