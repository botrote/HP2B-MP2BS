package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/Seo-A-Nam/MCNL/utils"
)

/*
*** # VALIDATOR.GO
***
****  This file mainly contains the functions that run validator to ask peer node for coordinate.
****  In this program, validator builds coordinate system, or mornitors network condition by doing self-cross-check on node0.
***
***  # What info does the message contain?
***
***   'message to peer' - it has info which is needed to calculate new coordinate of this peer node, including the evensignal number.
***     - eventSignal : 1 (calculate this peer node), 2 (sck this node), 3 (terminate this node)
***
****  'message from peer' - it contains the info of new coordinate (of this peer-node), Log, updated sign(mathematics) info of axis-node
 */
const (
	localUpdate uint16 = iota
	globalUpdate
)

var is_coord_done bool
var needLocalUpdate bool = false
var needGlobalUpdate bool = false

var nodeList []int

// BUILD THE WHOLE COORDINATE SYSTEM BY GETTING ALL COORDINATE OF PEER NODES
func build_CoordinateSystem(connList []*net.UDPConn) {
	ch1 := make(chan bool, utils.NodeTotal)

	// get coordinate of node i (= reference node)
	for i := 0; i < utils.Nb_Ref; i++ {
		go utils.GetPeerCoord(i, utils.Nb_Ref, utils.Ndim, connList, 1, ch1)
		<-ch1
		utils.NodeCount++
		utils.PrintCoords()
	}
	// get coordinate of node i+1 ~ k. (after the reference node)
	for i := utils.Nb_Ref; i < utils.NodeTotal; i++ {
		go utils.GetPeerCoord(i, utils.Nb_Ref, utils.Ndim, connList, 1, ch1)
	}
	for i := utils.Nb_Ref; i < utils.NodeTotal; i++ {
		<-ch1
		utils.NodeCount++
		utils.PrintCoords()
	}
	// print test log
	fmt.Println("COORDINATE ACCURACY =========================> ")
	fmt.Println(" estimated delay (from coordinate) / measured delay / error")
	utils.PrintTestResults()
	is_coord_done = true
	needGlobalUpdate = false
}

func build_LocalCoordinateSystem(connList []*net.UDPConn, nodeList []int) {
	fmt.Println("Local Coordinate Update Start", nodeList)
	utils.NodeCount -= len(nodeList)
	ch1 := make(chan bool, len(nodeList))

	for i := 0; i < len(nodeList); i++ {
		go utils.GetPeerCoord(nodeList[i], utils.Nb_Ref, utils.Ndim, connList, 1, ch1)
	}
	for i := 0; i < len(nodeList); i++ {
		<-ch1
		utils.PrintCoords()
	}
	fmt.Println("Local Coordinate Update Fin!")
	fmt.Println("COORDINATE ACCURACY =========================> ")
	fmt.Println(" estimated delay (from coordinate) / measured delay / error")
	utils.PrintTestResults()
	is_coord_done = true
	needLocalUpdate = false
}

func sendCoordInfo() { // tcp
	fmt.Println("sendCoordInfo start!")
	conn, err := net.Dial("tcp", utils.OmtreeIP+":"+utils.OmtreePortNum)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Conn fin!")

	msg := make([]byte, 4096)

	opt := 0
	binary.BigEndian.PutUint32(msg[0:4], uint32(utils.NodeTotal))
	binary.BigEndian.PutUint32(msg[4:8], uint32(utils.Ndim))
	binary.BigEndian.PutUint32(msg[8:12], uint32(opt))

	offset := 12

	for i := 0; i < utils.NodeTotal; i++ {
		binary.BigEndian.PutUint32(msg[offset:offset+4], uint32(i))
		offset += 4
		for j := 0; j < len(utils.NodeData[i].Coord); j++ {
			binary.BigEndian.PutUint32(msg[offset:offset+4], math.Float32bits(float32(utils.NodeData[i].Coord[j])))
			offset += 4
		}
	}
	conn.Write(msg)
	fmt.Println("sendCoordInfo to omtree fin!")

	// fmt.Println("----sendCoordInfo()----")
	// for i := 0; i < utils.NodeTotal; i++ {
	// 	if i == 0 || utils.NodeData[i].Coord[0] != 0 {
	// 		fmt.Printf("%d", i)
	// 		for j := 0; j < len(utils.NodeData[i].Coord); j++ {
	// 			fmt.Printf("%f", utils.NodeData[i].Coord[j])
	// 		}
	// 		fmt.Println("")
	// 	}
	// }
	// fmt.Println("----------------------------------------")
}

// EXECUTE SELF-CROSS-CHECK TO MONITOR NETWORK CONDITIONS
func monitor_origin(connList []*net.UDPConn) bool {
	// retrun false, if the delay between node0 and origin go over our setted thresh.
	var sck_error float64 = 0

	ch2 := make(chan bool)
	fmt.Println("**** START SELF-CROSS-CHECK ! ****")
	go utils.GetPeerCoord(0, utils.Nb_SCK_Ref, utils.Ndim, connList, 2, ch2)
	if <-ch2 {
		sck_error = 0
		for j := 0; j < utils.Ndim; j++ {
			if utils.NodeData[0].Coord[j] != 0 {
				sck_error += math.Pow(float64(utils.NodeData[0].Coord[j]), 2)
			}
		}
		sck_error = math.Sqrt(sck_error)
		fmt.Println("node 0 :", utils.NodeData[0].Coord)
		fmt.Printf("SCK result (delay between node0 and origin) : %f\n\t -- the smaller, the better.\n", sck_error)
		if sck_error > float64(utils.Origin_tresh) {
			return false // (= we have to re-implement our coordinate system, since the network condition has changed)
		}
	}
	return true
}

func handleSelfMaintenance() {
	udpAddr, err := net.ResolveUDPAddr("udp4", "172.17.0.3:8888")
	if err != nil {
		log.Fatal(err)
	}

	// udp endpoint를 파라미터로 넘기면 udp connection을 리턴한다.
	listen, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal(err)
	}

	// listen하고 있는 상태
	fmt.Println("handleSelfMaintenance listen")
	defer listen.Close()

	for {
		buffer := make([]byte, 2048) // Assuming int16 takes 2 bytes, and uint64 takes 8 bytes

		// udp connection으로 부터 값을 읽어들인다.
		_, addr, err := listen.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("readUDP from ", addr)

		updateType := uint16(binary.BigEndian.Uint16(buffer[0:2]))

		switch updateType {
		case globalUpdate:
			needGlobalUpdate = true
		case localUpdate:
			nodeNum := uint32(binary.BigEndian.Uint32(buffer[2:6]))
			offset := 6
			nodeList = nil
			for i := 0; uint32(i) < nodeNum; i++ {
				nd := uint32(binary.BigEndian.Uint32(buffer[offset : offset+4]))
				offset += 4
				nodeList = append(nodeList, int(nd))
			}
			needLocalUpdate = true
		}
	}
}

// ==========================  RUN VALIDATOR NODE   ==========================

func main() {
	utils.Ndim, _ = strconv.Atoi(os.Args[1])
	utils.Nb_Ref, _ = strconv.Atoi(os.Args[2])
	utils.Nb_SCK_Ref, _ = strconv.Atoi(os.Args[3])

	if utils.Ndim > utils.Nb_Ref {
		fmt.Println("[Error] Arguments : Invalid number of reference points or dimension!")
		return
	}

	fmt.Println("\n ================== START VALIDATOR NODE ================== ")
	fmt.Printf("The number of reference points >> %d\n", utils.Nb_Ref)
	fmt.Printf("the number of Dimension >> %d\n", utils.Ndim)

	utils.Initialize_database()
	connList := utils.DialAllPeers() // get udpConn pointers between peer and validator

	build_CoordinateSystem(connList)
	sendCoordInfo()

	go handleSelfMaintenance()

	var start time.Time

	build_LocalCoordinateSystem(connList, []int{3, 11, 19, 27, 35, 43, 51, 59, 67, 75}) //TODO: remove

	start = time.Now()
	for {
		if !(is_coord_done) {
			// if there is no coordinate system which is done, we can't process self-cross-check
			fmt.Println("validator - [error] we can't process self-cross-check, if there is no coordinate system done.")
			break
		}
		// time.Sleep(time.Duration(utils.SCKInterval) * time.Second)
		if needGlobalUpdate {
			fmt.Println("NETWORK CONDITION CHANGED.\n *** RE-IMPLEMENT THE COORDINATE SYSTEM !")
			utils.Reset_coordinate_data()
			is_coord_done = false
			utils.RESTART_COUNT++
			build_CoordinateSystem(connList)
			sendCoordInfo()
			start = time.Now()
		} else if needLocalUpdate {
			build_LocalCoordinateSystem(connList, nodeList)
		}

		end := time.Since(start)
		// fmt.Println("############", end, time.Duration(utils.SCKInterval)*time.Second)
		if end > time.Duration(utils.SCKInterval)*time.Second {
			fmt.Println("SCK Timer expired! ", end, time.Duration(utils.SCKInterval)*time.Second)
			err_flag := monitor_origin(connList)
			if !(err_flag) {
				fmt.Println("NETWORK CONDITION CHANGED.\n *** RE-IMPLEMENT THE COORDINATE SYSTEM !")
				utils.Reset_coordinate_data()
				is_coord_done = false
				utils.RESTART_COUNT++
				build_CoordinateSystem(connList)
				sendCoordInfo()
				start = time.Now()
			}
			start = time.Now()
		}
	}
	// TERMINATE ALL PEER NODES
	// utils.EndPeers(connList)

	defer fmt.Printf("THE NUMBER OF PEER ERROR : %d\n", utils.PEER_ERROR) // the number request for new refpoints from peer node.
	defer fmt.Printf("THE NUMBER OF RESTART : %d\n", utils.RESTART_COUNT) // the number of restart made by selfcrosscheck result
}
