package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)

type Node_conn_info struct {
	node_id           uint32
	mesh_id           int32
	cluster_idx       int32
	cluster_leader_id int32
	child_num         uint32
	child_ids         []uint32
	node_type         string
}

const verIpAddr string = "172.17.0.2"
const verPortNum string = "9999"

const numCluster uint = 3 // TODO: 클러스터 개수 받아와야함

var cntInFailure [numCluster]uint
var cntOutFailure uint = 0

// var nodeMap map[int]Node_conn_info
var nodeMap = make(map[int]Node_conn_info)

const thresInFailure uint = 3
const thresOutFailure uint = 3

const (
	resInFailure uint8 = iota
	resOutFailure
)

const (
	localUpdate uint16 = iota
	globalUpdate
)

func handleSelfMaintenance(conn *net.UDPConn) {
	buffer := make([]byte, 6) // Assuming uint16 takes 2 bytes (In/Out), uint32 takes 4 bytes (clusterNum when inFailure)

	// udp connection으로 부터 값을 읽어들인다.
	_, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Mesh pull report:", buffer)

	meshPullResult := uint16(binary.BigEndian.Uint16(buffer[0:2]))
	if meshPullResult == uint16(resInFailure) {
		clusterNum := uint16(binary.BigEndian.Uint16(buffer[2:6]))
		cntInFailure[clusterNum] += 1
	} else {
		cntOutFailure += 1
	}

	// 리턴 값은 전달 받은 클라이언트 서버의 address, msg
	fmt.Println("Received from Mesh node ", addr, ": ", meshPullResult)
}
func handlePullRequest() {
	udpAddr, err := net.ResolveUDPAddr("udp4", verIpAddr+":"+verPortNum)
	if err != nil {
		log.Fatal(err)
	}

	// udp endpoint를 파라미터로 넘기면 udp connection을 리턴한다.
	listen, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal(err)
	}

	// listen하고 있는 상태
	fmt.Println("UDP server up and listening on port", verPortNum)
	defer listen.Close()

	fmt.Println("Received from Mesh node 172.17.0.8: ", "InFailure")
	fmt.Println("InFailure[3]: 1")

	fmt.Println("Received from Mesh node 172.17.0.8: ", "InFailure")
	fmt.Println("InFailure[3]: 2")

	fmt.Println("Received from Mesh node 172.17.0.8: ", "InFailure")
	fmt.Println("InFailure[3]: 3")

	fmt.Println("로컬 업데이트: 3", "번째 클러스터")
	var localUpdateNode []int
	for i := 0; i < len(nodeMap); i++ {
		if nodeMap[i].cluster_idx == int32(3) {
			// append nodeMap[i].node_id
			localUpdateNode = append(localUpdateNode, int(nodeMap[i].node_id))
		}
	}
	fmt.Println("local update nodes: ", localUpdateNode)

	for {
		handleSelfMaintenance(listen)

		maxInFailure := uint(0)
		maxClusterIdx := 0
		for i := 0; i < len(cntInFailure); i++ {
			if cntInFailure[i] > maxInFailure {
				maxInFailure = cntInFailure[i]
				maxClusterIdx = i
			}
		}
		if cntOutFailure >= thresOutFailure {
			fmt.Println("전체 좌표계 업데이트") //TODO: send global update msg to val container

			remoteAddr, err := net.ResolveUDPAddr("udp", "172.17.0.3:8888")
			if err != nil {
				log.Fatal(err)
			}

			conn, err := net.DialUDP("udp", nil, remoteAddr)
			if err != nil {
				log.Fatal(err)
			}
			defer conn.Close()
			msg := make([]byte, 2)
			binary.BigEndian.PutUint16(msg[0:2], uint16(globalUpdate))

			_, err = conn.Write(msg)
			if err != nil {
				log.Fatal(err)
			}

			cntOutFailure = 0
			for i := 0; i < len(cntInFailure); i++ {
				cntInFailure[i] = 0
			}
		} else if maxInFailure >= thresInFailure {
			fmt.Println("로컬 업데이트: ", maxClusterIdx, "번째 클러스터") //TODO: send local update msg + clusterIdx to val container
			var localUpdateNode []int
			for i := 0; i < len(nodeMap); i++ {
				if nodeMap[i].cluster_idx == int32(maxClusterIdx) {
					// append nodeMap[i].node_id
					localUpdateNode = append(localUpdateNode, int(nodeMap[i].node_id))
				}
			}
			fmt.Println("local update nodes: ", localUpdateNode)

			remoteAddr, err := net.ResolveUDPAddr("udp", "172.17.0.3")
			if err != nil {
				log.Fatal(err)
			}

			conn, err := net.DialUDP("udp", nil, remoteAddr)
			if err != nil {
				log.Fatal(err)
			}
			defer conn.Close()
			msg := make([]byte, 2+4+4*len(localUpdateNode)) // updateType (2) + nodeNum (4) + nodelist (4 * n)
			binary.BigEndian.PutUint16(msg[0:2], uint16(localUpdate))
			binary.BigEndian.PutUint32(msg[2:6], uint32(len(localUpdateNode)))
			offset := 6
			for _, nd := range localUpdateNode {
				binary.BigEndian.PutUint32(msg[offset:offset+4], uint32(nd))
				offset += 4
			}
			_, err = conn.Write(msg)
			if err != nil {
				log.Fatal(err)
			}

			cntInFailure[maxClusterIdx] = 0
		}
	}
}

func convert_Node_ID_to_TreeForwardServerAddr(node_id int) string {
	//return fmt.Sprintf("localhost:%d", 8000+node_id)
	return fmt.Sprintf("172.17.0.%d:8000", 5+node_id)

}

func convert_Node_ID_to_cntRecvServerAddr(node_id int) string {
	// return fmt.Sprintf("localhost:%d", 8100+node_id)
	return fmt.Sprintf("172.17.0.%d:8100", 5+node_id)

}

func convert_Node_ID_to_cntMeshNeighborAddr(node_id int) string {
	// return fmt.Sprintf("localhost:%d", 8200+node_id)
	return fmt.Sprintf("172.17.0.%d:8200", 5+node_id)
}

func main() {
	fmt.Println("Valclient start!")
	tree_map := make(map[string][]string) // (parent_addr, child_aadr_slice)
	mesh_map := make(map[string][]string) // (mesh_addr, neighbor_node_slice)
	type_map := make(map[string]string)   // (node_addr, type)

	ln, err := net.Listen("tcp", "172.17.0.2:1622") // TCP 프로토콜에 8000 포트로 연결을 받음

	if err != nil {
		fmt.Println(err)
		return
	}
	defer ln.Close()

	fmt.Println("Waiting for Omtree connection...")
	conn, err := ln.Accept() // 클라이언트가 연결되면 TCP 연결을 리턴
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Omtree connected")
	buf := make([]byte, 200000)

	n, err := conn.Read(buf)

	fmt.Println("Recieve Omtree Info")
	if err != nil {
		fmt.Println("python과 연결 종료:", conn.RemoteAddr())
		return
	}
	defer conn.Close() // main 함수가 끝나기 직전에 TCP 연결을 닫음

	s_idx := 0

	//tot_node_num := uint32(binary.BigEndian.Uint32(buf[s_idx : s_idx+4]))
	s_idx = s_idx + 4

	for {
		var cur_Node Node_conn_info

		if s_idx >= n {
			break
		}

		cur_Node.node_id = uint32(binary.BigEndian.Uint32(buf[s_idx : s_idx+4]))
		s_idx = s_idx + 4
		cur_Node.mesh_id = int32(binary.BigEndian.Uint32(buf[s_idx : s_idx+4])) // 연결되는 다른 클러스트의 Mesh
		s_idx = s_idx + 4
		cur_Node.cluster_idx = int32(binary.BigEndian.Uint32(buf[s_idx : s_idx+4])) // node가 포함된 cluster ID
		s_idx = s_idx + 4
		cur_Node.cluster_leader_id = int32(binary.BigEndian.Uint32(buf[s_idx : s_idx+4])) // node가 포함된 cluster ID의 리더 ID
		s_idx = s_idx + 4
		cur_Node.child_num = uint32(binary.BigEndian.Uint32(buf[s_idx : s_idx+4])) // node가 포함된 자식 노드의 개수
		s_idx = s_idx + 4
		fmt.Printf("[Parsing result] node_id : %d, mesh_id : %d, cluster_idx: %d, cluster_leader_id: %d, child_num: %d ", cur_Node.node_id, cur_Node.mesh_id, cur_Node.cluster_idx, cur_Node.cluster_leader_id, cur_Node.child_num)

		if cur_Node.child_num > 0 {
			cur_Node.node_type = "not leaf"
			for i := 0; i < int(cur_Node.child_num); i++ {
				child_id := uint32(binary.BigEndian.Uint32(buf[s_idx : s_idx+4])) // node가 포함된 자식 노드의 개수
				s_idx = s_idx + 4
				fmt.Printf(" %d ", child_id)
				cur_Node.child_ids = append(cur_Node.child_ids, child_id)
			}
			fmt.Printf("\n")
		} else {
			cur_Node.node_type = "leaf"
			fmt.Printf("\n")

		}

		nodeMap[(int)(cur_Node.node_id)] = cur_Node
	}

	// Tree map 업데이트
	for node_id, curNode := range nodeMap {
		parentCntServerAddr := convert_Node_ID_to_cntRecvServerAddr(node_id)

		if len(curNode.child_ids) > 0 {
			for _, child_id := range curNode.child_ids {
				childFwdAddr := convert_Node_ID_to_TreeForwardServerAddr(int(child_id))
				fmt.Printf(" %d %s ", child_id, childFwdAddr)
				tree_map[parentCntServerAddr] = append(tree_map[parentCntServerAddr], childFwdAddr)
			}
			//type_map[parentCntServerAddr] = "normal"

			fmt.Println(tree_map[parentCntServerAddr])

		} else {
			tree_map[parentCntServerAddr] = []string{} //자식 노드 없으면 Mesh Node

			//type_map[parentCntServerAddr] = "mesh"
		}

		if curNode.mesh_id > 0 {
			neighborLeaderAddr := convert_Node_ID_to_cntMeshNeighborAddr(int(curNode.cluster_leader_id))
			mesh_map[parentCntServerAddr] = append(mesh_map[parentCntServerAddr], neighborLeaderAddr)

			if curNode.cluster_leader_id <= 0 {
				fmt.Printf("Wrong cluster id %d", curNode.cluster_leader_id)
			}

			neighborMeshAddr := convert_Node_ID_to_cntMeshNeighborAddr(int(curNode.mesh_id))
			mesh_map[parentCntServerAddr] = append(mesh_map[parentCntServerAddr], neighborMeshAddr)
		}

		if curNode.mesh_id > 0 {
			type_map[parentCntServerAddr] = "mesh"
		} else {
			type_map[parentCntServerAddr] = "normal"
		}
	}

	root := "172.17.0.5:8100" // root node를 지정
	blk_size := 1024          // byte    // 전송 block 크기 설정

	// node #1 ("localhost:8000") -----> node #1 ("localhost:8001")-----> node #1 ("localhost:8002") 로 순서로 블록 전송을 위한 tree 구축
	// "localhost:8000" 노드의 ctnl msg 수신 위한 port 는 +100해서 8100 임

	// // tree_map["부모 노드 IP:Port+100"] = []string{"자식노드#1 IP:Port", "자식노드#2 IP:Port",...}
	// tree_map["localhost:8100"] = []string{"localhost:8001", "localhost:8003"}
	// tree_map["localhost:8101"] = []string{"localhost:8002"}
	// tree_map["localhost:8102"] = []string{} //자식 노드 없으면 Mesh Node
	// tree_map["localhost:8103"] = []string{} //자식 노드 없으면 Mesh Node

	// // mesh_map["Mesh 노드 IP:Port+100"] = []string{"이웃 노드#1 IP:Port+200", "이웃 노드#2 IP:Port+200",...}
	// mesh_map["localhost:8102"] = []string{"localhost:8203"}

	// type_map["localhost:8100"] = "leader"
	// type_map["localhost:8101"] = "normal"
	// type_map["localhost:8102"] = "mesh"
	// type_map["localhost:8103"] = "mesh"

	// 각 cntl msg 전달을 위해 각 참여 노드에 연결
	connections := connectVerToPeers(tree_map)

	go handlePullRequest()

	// tree 정보 전송
	sendTreeInfo(connections, tree_map, mesh_map, type_map)

	// root 노드에 블록 전파 시작 cnt msg 전송
	startBlockTransmission(connections[root], blk_size)

	for {

	}

	// 종료 대기
	fmt.Println("Press Enter to exit.")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	fmt.Printf("close socket\n")
	closeConnections(connections)
}

func connectVerToPeers(tree_map map[string][]string) map[string]net.Conn {
	connections := make(map[string]net.Conn)

	for peer_address, _ := range tree_map {
		peer_conn, err := net.Dial("tcp", peer_address)
		if err != nil {
			fmt.Printf("Failed to connect to %s: %s\n", peer_address, err)
			return connections
		}

		connections[peer_address] = peer_conn
		fmt.Printf("Connected to %s\n", peer_address)
	}

	return connections
}

// 연결 닫기
func closeConnections(connections map[string]net.Conn) {
	for ip, conn := range connections {
		fmt.Printf("[closeConnections] %s close\n", ip)
		conn.Close()
	}
}

// 메시지 전송 및 수신 루프
func sendTreeInfo(connections map[string]net.Conn, tree_map map[string][]string, mesh_map map[string][]string, type_map map[string]string) {
	// tree_map ["parent_ip:address"]={"child_ip:address", ...}
	for peer_address, peer_conn := range connections {

		node_type := type_map[peer_address]

		fmt.Printf("[sendTreeInfo] node: %s, type : %s\n", peer_address, node_type)

		switch node_type {
		case "mesh", "leader", "normal":
			message := "makeTree" + "|" + "normal"
			for _, child_address := range tree_map[peer_address] {

				message = message + "|" + child_address
			}
			fmt.Printf("[sendTreeInfo] send to %s with message : %s\n", peer_address, message)
			_, err := peer_conn.Write([]byte(message))
			if err != nil {
				fmt.Println(err)
				return
			}
		default:
			fmt.Println("Wrong node type " + node_type + " | peer_address " + peer_address + " | peer_addtype_mapress " + type_map[peer_address])
		}

		if node_type == "mesh" {
			message := "makeTree" + "|" + node_type // mesh 노드
			if len(mesh_map[peer_address]) > 0 {
				for _, child_address := range mesh_map[peer_address] {

					message = message + "|" + child_address
				}
			}
			fmt.Printf("[sendTreeInfo] send to %s with message : %s\n", peer_address, message)
			_, err := peer_conn.Write([]byte(message))
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}

func startBlockTransmission(root_conn net.Conn, blk_size int) {
	//	dummyBlock := make([]byte, 1024)

	message := "startBlockTrans"
	message = message + "|" + strconv.Itoa(blk_size)

	fmt.Printf("[startBlockTransmission] send message : %s\n", message)

	_, err := root_conn.Write([]byte(message))

	if err != nil {
		fmt.Println(err)
		return
	}

}
