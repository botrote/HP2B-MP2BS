package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"time"
)

// 구현 완료 후 TODO 별도의 package로 분리 필요

// 서버에 연결
func hp2b_client_connectToServers(servers []string) map[string]net.Conn {
	connections := make(map[string]net.Conn)

	for _, child_address := range servers {
		// fmt.Printf("["+peer_name+"]child_address %s\n", child_address)

		child, err := net.Dial("tcp", child_address)
		if err != nil {
			fmt.Printf("Failed to connect to %s: %s\n", child_address, err)
			return connections
		}

		connections[child_address] = child
		fmt.Printf("["+peer_name+"] Connected to %s\n", child_address)
	}

	return connections
}

// 연결 닫기
func hp2b_client_closeConnections(connections map[string]net.Conn) {
	for ip, conn := range connections {
		fmt.Printf("[closeConnections] %s close\n", ip)
		conn.Close()
	}
}

// 메시지 전송 및 수신 루프
func sendAndReceiveLoop(connections map[string]net.Conn) {
	for name, child := range connections {
		go sendAndReceiveMessages(child, name)
	}
}

// 메시지 전송 및 수신 처리
func sendAndReceiveMessages(child net.Conn, serverName string) {
	reader := bufio.NewReader(child)

	// 서버로부터 메시지 수신
	go func() {
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Error reading from %s: %s\n", serverName, err)
				return
			}

			fmt.Printf("[%s] Received: %s", serverName, message)
		}
	}()

	// 메시지 전송
	go func() {
		i := 0
		for {
			message := "Hello " + serverName + " " + strconv.Itoa(i)
			_, err := child.Write([]byte(message + "\n"))
			if err != nil {
				fmt.Println(err)
				return
			}
			i++
			time.Sleep(1 * time.Second)
		}
	}()
}
