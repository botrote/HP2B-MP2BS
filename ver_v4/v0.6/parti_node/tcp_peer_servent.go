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
)

type ClientSession struct {
	conn   net.Conn
	active bool
}

var connections map[string]net.Conn
var peer_name string
var blk_header_len = 64

func forwardBlockForAllChild(child_cons map[string]net.Conn, buffer []byte) {
	for child_addr, child_con := range connections {
		fmt.Printf("["+peer_name+"] send from %s block to %s \n", peer_name, child_addr)
		go ForwardBlock(child_con, child_addr, buffer)
	}
}

func ForwardBlock(child net.Conn, serverName string, buffer []byte) {

	_, err := child.Write(buffer)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// 부모 노드 (client) 에 block 수신 및 자식 노드 (sever)로 블록 forwarding
func handleClient(session *ClientSession) {
	defer session.conn.Close()

	buffer := make([]byte, 1024*100)

	for {
		n, err := session.conn.Read(buffer)
		if err != nil {
			fmt.Println("클라이언트 연결 종료:", session.conn.RemoteAddr())
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

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("서버 시작 실패:", err)
		done <- true
		return
	}
	defer listener.Close()

	fmt.Println("서버 시작")

	for {
		select {
		case <-done:
			fmt.Println("프로그램 종료")
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("클라이언트 연결 수락 실패:", err)
				continue
			}

			session := &ClientSession{conn: conn, active: true}
			mutex.Lock()
			fmt.Println("[" + peer_name + "] 부모 노드 " + conn.RemoteAddr().String() + " 와 연결")
			clients[conn.RemoteAddr().String()] = session
			mutex.Unlock()

			go handleClient(session)
		}
	}
}

// 부모 노드 (client) 와 연결
func peerServerForMissingBlock(ip, port string, wg *sync.WaitGroup, done chan bool) {
	defer wg.Done()

	clients := make(map[string]*ClientSession)
	var mutex sync.Mutex

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("서버 시작 실패:", err)
		done <- true
		return
	}
	defer listener.Close()

	fmt.Println("Pull response를 위한 서버 시작")

	for {
		select {
		case <-done:
			fmt.Println("프로그램 종료")
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("클라이언트 연결 수락 실패:", err)
				continue
			}

			session := &ClientSession{conn: conn, active: true}
			mutex.Lock()
			fmt.Println("[" + peer_name + "] 이웃 노드 " + conn.RemoteAddr().String() + " 와 연결")
			clients[conn.RemoteAddr().String()] = session
			mutex.Unlock()

			go handleMissingBlock(session)
		}
	}
}

func requestAndReceiveMissingBlock(peer_address string, blk_num uint64) {

	peer_conn, err := net.Dial("tcp", peer_address)

	if err != nil {
		fmt.Printf("Failed to connect to %s: %s\n", peer_address, err)
		return
	}
	defer peer_conn.Close()

	message := "requestBlock"
	message = message + "|" + strconv.FormatUint(blk_num, 10)
	peer_conn.Write([]byte(message))

	buffer := make([]byte, 1024*100)

	n, err := peer_conn.Read(buffer)
	if err != nil {
		fmt.Println("클라이언트 연결 종료:", peer_conn.RemoteAddr())
		return
	}
	data := string(buffer[:n])
	recv_blk_num := uint64(binary.BigEndian.Uint64(buffer[0:blk_header_len]))

	if recv_blk_num == myNode.curBlkLength+1 {
		myNode.curBlkLength = blk_num
		fmt.Printf("["+peer_name+"] Pull 요청으로 수신된 데이터 [%s]: %d %s\n", peer_conn.RemoteAddr(), recv_blk_num, data)
	}
}

func sendMissingBlockToAll(peer_address string, blk_num uint64) {

	// 구현 해야 함.

	// peer_conn, err := net.Dial("tcp", peer_address)
	// if err != nil {
	// 	fmt.Printf("Failed to connect to %s: %s\n", peer_address, err)
	// 	return
	// }
	// defer peer_conn.Close()

	// message := "sendMissingBlock"
	// message = message + "|" + strconv.FormatUint(blk_num, 10)
	// peer_conn.Write([]byte(message))

}

// Pull request에 대한 송신 수행
func handleMissingBlock(session *ClientSession) {
	defer session.conn.Close()

	buffer := make([]byte, 1024*100)

	for {
		n, err := session.conn.Read(buffer)
		if err != nil {
			fmt.Println("Minsing Block 송수신을 위한 연결 종료:", session.conn.RemoteAddr())
			break
		}

		data := string(buffer[:n])
		//fmt.Printf("["+peer_name+"] 검증노드에서 수신된 데이터 [%s]: %s\n", session.conn.RemoteAddr(), data)

		// 문자열을 "|" 기준으로 분리하여 슬라이스에 저장
		parts := strings.Split(data, "|")

		if parts[0] == "requestBlock" {
			req_blk_idx, _ := strconv.ParseUint(parts[1], 10, 64)
			if req_blk_idx <= myNode.curBlkLength {
				// 요청 받은 블록 전송

				// TODO blk_size 를 노드 정보에 들어가에 해야 함.
				blk_size := 1024
				block := make([]byte, blk_size)

				msg := make([]byte, blk_header_len+blk_size)

				// 헤더 값 삽입
				binary.BigEndian.PutUint64(msg[0:blk_header_len], uint64(req_blk_idx))

				// 블록 데이터 복사
				copy(msg[blk_header_len:], block)
				session.conn.Write(msg)
			}
		} else {
			fmt.Printf("Invalid message format %s\n", data)
		}

	}
}

func meshUpdate() {
	meshNeedUpdate = true

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

// 검증 노드 (verification node) 와 Message 송/수신
func handleVerNode(session *ClientSession) {
	defer session.conn.Close()

	buffer := make([]byte, 1024)

	for {
		n, err := session.conn.Read(buffer)
		if err != nil {
			fmt.Println("클라이언트 연결 종료:", session.conn.RemoteAddr())
			break
		}

		data := string(buffer[:n])
		fmt.Printf("["+peer_name+"] 검증노드에서 수신된 데이터 [%s]: %s\n", session.conn.RemoteAddr(), data)

		// 문자열을 "|" 기준으로 분리하여 슬라이스에 저장
		parts := strings.Split(data, "|")
		var neighborAddresses []string

		// 첫 번째 항목이 "makeTree"인 경우에만 슬라이스에 저장
		if parts[0] == "makeTree" {
			meshNeedUpdate = true
			if parts[1] == "leader" || parts[1] == "normal" {
				childAddresses := parts[2:]
				fmt.Printf("["+peer_name+"] %d children, update isMesh to false\n", len(childAddresses))
				myNode.isMesh = false

				fmt.Printf("["+peer_name+"] slice len %d connections len %d\n", len(childAddresses), len(connections))
				if len(connections) != 0 {
					hp2b_client_closeConnections(connections)
				}

				connections = hp2b_client_connectToServers(childAddresses)
			} else { //parts[1] == "mesh"
				fmt.Printf("[" + peer_name + "] no children, update isMesh = true and neighbors\n")
				if parts[2] == "mesh" {
					neighborAddresses = parts[3:]
				} else {
					neighborAddresses = parts[2:]
				}
				fmt.Println("@@@", neighborAddresses)
				if len(neighborAddresses) > 0 {
					addMeshNeighbor(neighborAddresses, []uint8{nodeTypeClusterLeader, nodeTypeMesh})
				}

				myNode.isMesh = true
				meshNeedUpdate = false
				go MeshRun()
			}

			// 첫 번째 항목이 "startBlockTrans"인 경우에만 블록을 생성하고 전송
		} else if parts[0] == "startBlockTrans" {
			fmt.Printf("root node generate block with %s (byte) \n", parts[1])
			blk_size, _ := strconv.Atoi(parts[1])

			genBlock(blk_size)

			// block := buffer[:blk_size]
			// if len(connections) != 0 {
			// 	forwardBlockForAllChild(connections, block)
			// }

		} else {
			fmt.Printf("Invalid message format %s\n", data)
		}

		//response := "서버 응답: 받았습니다!"
		//session.conn.Write([]byte(response))
	}
}

// 검증 노드 (Verificaiton node) 와 연결
func peerVerServer(ip, port string, wg *sync.WaitGroup, done chan bool) {
	defer wg.Done()

	clients := make(map[string]*ClientSession)
	var mutex sync.Mutex

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("서버 시작 실패:", err)
		done <- true
		return
	}
	defer listener.Close()

	fmt.Println("[" + peer_name + "] 검증노드와 통신을 위한 서버 실행")

	for {
		select {
		case <-done:
			fmt.Println("[" + peer_name + "] 검증노드 연결 종료")
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("["+peer_name+"] 검증노드 연결 수락 실패:", err)
				continue
			}

			session := &ClientSession{conn: conn, active: true}
			mutex.Lock()
			fmt.Println(conn.RemoteAddr().String())
			clients[conn.RemoteAddr().String()] = session
			mutex.Unlock()

			go handleVerNode(session)
		}
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
