package utils

import (
	"fmt"
	"log"
	"net"
	"google.golang.org/protobuf/proto"
)

/*
*** # VALIDATOR_UTILS_NETWORK.GO
***
****  This file mainly contains the functions that directly communicate with peer via network.
****  those functions create/destroy udp-connection pointers, or they exchange message with peer. 
***
*/


// =========== GENERATE OR TERMINATE THE UDP CONNECTION POINTER ===========

// DIAL TO PEER NODES AND RETURN IT'S UDP CONNECTION POINTERS
func DialAllPeers() []*net.UDPConn {
	connList := make([]*net.UDPConn, 0)

	for i := 0; i < NodeTotal; i++ {
		udpAddr, err := net.ResolveUDPAddr("", NodeData[i].NodeIp)
		if err != nil {
			log.Fatal(err)
		}
		conn, err := net.DialUDP("udp", nil, udpAddr)
		if err != nil {
			log.Println("[Network Error] Connect to udp server failed,err:", err)
		}
		connList = append(connList, conn)
	}
	fmt.Println("   ************ Finished Dialing all nodes ************   ")
	return connList
}

// SEND TERMINATE SIGNAL TO PEERS WHEN THE PROCESS IS ALL DONE
func EndPeers(connList []*net.UDPConn) {
	fmt.Println("   ************ TERMINATE ALL PEER NODES ************   ")
	for i := 0; i < NodeTotal; i++ {
		in := &TData{EventSig: 3}
		sendMessage_to_peer(in, connList[i])
	}
}

// =================== COMMUNICATE WITH PEER VIA UDP =====================

// SEND PROTO MESSAGE TO PEER VIA UDP
func sendMessage_to_peer(in *TData, conn *net.UDPConn) {
	data, err := proto.Marshal(in)
	if err != nil {
		log.Fatal("[Proto Error] marshalling error: ", err)
	}
	buf := []byte(data)
	_, err = conn.Write(buf)
	if err != nil {
		log.Fatal("[Network Error] Send data failed, err:", err)
	}
	fmt.Printf("** validator sent message to node%d\n", in.MyId)
}

// RECIEVE PROTO MESSAGE FROM PEER VIA UDP
func recieveMessage_from_peer(conn *net.UDPConn, is_sck bool) ([]float32, bool) {
	fmt.Println("Waiting for Peer ...")
	result := make([]byte, 2048)
	res := new(PeerResponse)
	n, _, err := conn.ReadFromUDP(result) // get message from peer
	if err != nil {
		log.Fatal("[Network Error] Read from udp server failed ,err:", err)
		return nil, true
	}
	err = proto.Unmarshal(result[0:n], res)
	if err != nil {
		log.Fatal("[Proto Error] unmarshalling error: ", err)
		return nil, true
	}
	fmt.Println(res.Log.Peerlog)
	if !is_sck {
		if res.Log.GetErrFlag() {
			PEER_ERROR++
			return nil, true
		}
		if res.Log.Sign != nil {
			fmt.Println("Change sign of coord [before] :")
			for i, _ := range res.Log.Sign {
				fmt.Println(NodeData[i].Coord)
			}
			fmt.Println("Change sign of coord [after] :")
			for i, s := range res.Log.Sign {
				fmt.Println(NodeData[i].Coord)
				if s == false {
					NodeData[i+1].Coord[i] *= -1
				}
			}
		}
		TestData[NodeCount] = [2]float32{res.Test[0], res.Test[1]}
	}
	fmt.Println("validator got new Coordinate: ", res.Node)
	return res.Node, false
}
