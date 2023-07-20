package main

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/Seo-A-Nam/MCNL/utils"
	"google.golang.org/protobuf/proto"
)

/*
*** PEERNODE.GO
***
***  this file mainly contains the functions that run peernode to communicate with validator node.
***
***  # What info does the message contain?
***
***   'message from validator' - it has info which is needed to calculate new coordinate of this peer node, including the evensignal number.
***     - eventSignal : 1 (calculate this peer node), 2 (sck this node), 3 (terminate this node)
***
****  'message to validator' - it contains the info of new coordinate (of this peer-node), Log, updated sign(mathematics) info of axis-node
***
 */

// GET MESSAGE FROM VALIDATOR NODE
func recieveMessage_from_validator(udpConn *net.UDPConn) (*utils.TData, *net.UDPAddr) {
	data := make([]byte, 1024)
	n, remoteAddr, err := udpConn.ReadFromUDP(data) // get info required to find my coordinatef
	if err != nil {
		log.Fatal("[Network Error] Listen failed", err)
	}
	log.Println("node recieved msg from validator!")
	in := &utils.TData{}
	err = proto.Unmarshal(data[0:n], in)
	if err != nil {
		log.Fatal("[Proto Error] unmarshalling error: ", err)
	}
	fmt.Println("RECIEVED A NEW MESSAGE FOR", in.MyId, "\nEventSignal :", in.EventSig)
	return in, remoteAddr
}

//  SEND MESSAGE TO VALIDATOR NODE
func sendMessage_to_validator(in *utils.TData, remoteAddr *net.UDPAddr, udpConn *net.UDPConn, is_sck bool) {
	var err_flag bool = false
	myCoord, err, Log, sign, test := utils.GetMyCoord(in, is_sck) // get my coordinate
	if err != nil {
		log.Println("peer - [error] There was error when caculating coordinates : ", err)
		err_flag = true
	}
	fmt.Println("nodes sends the message to validator!")
	sig := utils.PeerSignal{ErrFlag: err_flag, Peerlog: Log, Sign: sign}
	msg := utils.PeerResponse{Log: &sig, Node: myCoord, Test: test}

	result, err := proto.Marshal(&msg)
	if err != nil {
		log.Fatal("peer - [Proto Error] marshalling error: ", err)
	}
	buf := []byte(result)
	_, err = udpConn.WriteToUDP(buf, remoteAddr) // send message : peer -> validator
	if err != nil {
		log.Fatal("peer - [Network Error] write to udp server failed,err:", err)
	}
}

// ================================ RUN PEER NODE ================================

// EVENT HANDLER : EXECUTE APPROPRIATE PROCESS WHEN THIS PEERNODE GOT A MESSAGE FROM VALIDATOR,
func peerNode(udpConn *net.UDPConn, myIp string) {
	var is_node bool = false
	var is_sck bool = true
	var eventSig int = 0

	// EVENT SIGNAL : 1 (calculate this peer node), 2 (sck this node), 3 (terminate this node)
	for {
		// ============== RECIEVE NECCESARY INFO AND EVENT SIGNAL FOR COORD CALCULATION ==============
		in, remoteAddr := recieveMessage_from_validator(udpConn)
		eventSig = int(in.GetEventSig())
		//log.Printf("node(%s) recieved eventSig : %d\n", myIp, eventSig)
		switch eventSig {
		case 1: // calculate node coordinate
			sendMessage_to_validator(in, remoteAddr, udpConn, is_node)
		case 2: // selfCrossCheck this node(= node0)
			sendMessage_to_validator(in, remoteAddr, udpConn, is_sck)
		case 3: // terminate this node
			log.Println("peer - Terminate the node")
			return
		default:
			log.Println("peer - [error] Undefined event signal.")
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
	log.Println("=============== STARTING PEER NODE ===============\n")
	myIp := get_local_ip()
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 1053}) // make a listener pointer of UDP Connection
	if err != nil {
		log.Fatal("[Network Error] Listen failed", err)
	}
	peerNode(udpConn, myIp)
}
