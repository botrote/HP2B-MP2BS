package utils

import (
	"fmt"
	"math/rand"
	"net"
)

/*
*** # VALIDATOR_UTILS_GETCOORD.GO
***
****  This file mainly contains the functions that can be used to get coordinate of a peer.
****  the functions make validaotr pick reference points and send those info to a peer, to get the peer's coordinate info(+ etc) as a message.
***
***  # REFERENCE POINT SELECTION
****   - originary peer node : should always include node0 for refpoint
****   - selfcrosscheck for node0 : should not include node0 for refpoint
 */

// ================== GET INFO OF REFERENCE NODES ==================

// GET INFO(COORDINATE + IPADDRESS) OF REFERECNE POINTS
func getReferencePoints(in *TData, ref_nb int, is_sck bool) {
	// set the number of reference points
	var start int = 0
	if in.MyId == 0 {
		if is_sck {
			start = 1
		} else { // peer node0 doesn't need any neighbor to get it's coordinate.
			return
		}
	}
	nb := (ref_nb + start)
	if ref_nb > NodeCount {
		nb = NodeCount // ref_nb cannot be over the number of nodes we already have in coordinate system.
	}
	for i := start; i < nb; i++ {
		in.RefPoints = append(in.RefPoints, &NodeData[i])
	}
}

// ================== GET COORDINATE OF A PEER NODE ==================

// REQUEST PEER FOR A COORDINATE AND GET THAT COORDINATE TO SAVE IT
func GetPeerCoord(id int, ref_nb int, ndim int, connList []*net.UDPConn, eventSig int, ch chan bool) {
	var is_sck bool = false

	fmt.Printf("\n   ************ GET COORDINATE OF PEER NODE[%d] ************   \n\n", id)
	if eventSig == 2 {
		is_sck = true
	}
	// SEND MESSAGE TO PEER (INFO OF REFERENCE POINTS)
	node := make([]*NodeInfo, 0)
	in := &TData{
		Ndim: int32(ndim), EventSig: int32(eventSig), MyId: int32(id), RefPoints: node,
	}
	getReferencePoints(in, ref_nb, is_sck) // !!! get reference points !!!!
	PrintRefpoints(in)
	sendMessage_to_peer(in, connList[id])
	res, err_flag := recieveMessage_from_peer(connList[id], is_sck)

	// ** ERROR HANDLING : we cannot find coordinate of peer node.
	if err_flag {
		if !(is_sck) {
			n := (rand.Intn(NodeTotal-id-1) + (id + 1))
			// !!! pick other node as node[k] to get node[k] -- at this mement, 'the other node' should be the one whose coordinate isn't determined yet. !!!
			fmt.Printf("** Swapping Node %d and Node %d\n", id, n)
			fmt.Printf("before[%d] : %s \\ after[%d]: %s\n", NodeData[id].Id, NodeData[id].NodeIp, NodeData[n].Id, NodeData[n].NodeIp)
			NodeData[id].NodeIp, NodeData[n].NodeIp = NodeData[n].NodeIp, NodeData[id].NodeIp
			connList[id], connList[n] = connList[n], connList[id]
			fmt.Printf("after[%d] : %s \\ before[%d]: %s\n\n", NodeData[id].Id, NodeData[id].NodeIp, NodeData[n].Id, NodeData[n].NodeIp)

		}
		GetPeerCoord(id, ref_nb, ndim, connList, eventSig, ch)
		return
	}
	// RECIEVE COORDINATE DATA AND SAVE IT
	copy(NodeData[id].Coord, res)
	ch <- true // send signal that this function is done.
}

// =================== PICK RANDOM NODES ==========================

// func find_index_of_n(arr []int, n int) int {
// 	for i, x := range arr {
// 		if x == n {
// 			return i
// 		}
// 	}
// 	return len(arr)
// }

// func pick_rand_N_nb(n int, max int, zero_flag bool) []int {
// 	// pick n numbers in range of 0~max,
// 	// if zero_flag is true, the number should include zero
// 	if n > max {
// 		log.Println("[error] Wrong parameter : Can't pick N numbers in this range!\n")
// 		return nil
// 	}
// 	tmp := rand.Perm(max) // shuffle numbers from 0 ~ max
// 	idx_z := find_index_of_n(tmp, 0)
// 	arr := append(tmp[:idx_z], tmp[(idx_z+1):]...) // pop 0 from the shuffled array
// 	if zero_flag {
// 		arr = append([]int{int(0)}, arr...)
// 		//return arr[:n-1]
// 	}
// 	//fmt.Println(arr)
// 	return arr[:n]
// }
