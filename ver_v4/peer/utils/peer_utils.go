package utils

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

/*
*** PEER_UTILS_COORD.GO
**** this file mainly contains the function that bring new coordinate of this peer node.
 */

/*
*** "PREREQUISITE INFO" TO GET COORDINATE OF NEW PEER NODE
****  1. coordinate of reference points. 2. ipaddress of the reference points, in order to get delay(avgRTT) from this node.
 */

// GET AVERAGE RTT TO THE IPADDRESS
func GetAvgRTT(ipaddr string) (float64, error) {
	var avgRtt float64                                                      // agvRtt of ping 1~3 (ping 0 is excluded)
	cmd := exec.Command("ping", ipaddr, "-c", "4", "-i", "1.0", "-s", "16") // send 4 pings _ time interval : 0.2s
	output, err := cmd.CombinedOutput()
	if err != nil {
		return -1, errors.New("Can't get avgRtt")
	}
	fmt.Println("output: ", string(output))

	line := strings.Split(string(output), "\n")
	// fmt.Println(len(line))

	fmt.Println("ping to ", ipaddr)
	var cnt float64
	for i := 2; i <= 4; i++ {
		words := strings.Split(line[i], " ")
		if len(words) > 1 {
			fmt.Println(line[i], "words:", words)

			s, err := strconv.ParseFloat(words[6][5:], 64) // extract ping rtt
			if err != nil {
				return -1, errors.New("Can't convert string to float32")
			}
			// fmt.Println(line[i], "rtt=", s)
			avgRtt += s
			cnt += 1
		}
	}
	avgRtt /= cnt
	return avgRtt, nil
}

// RETRUN NEW COORDINATE OF THIS NODE BY CALLING THE CALCULATION FUNCTION AFTER GETTING DELAY(RTT) VECTORS
func GetMyCoord(myNode *TData, is_sck bool) ([]float32, error, string, []bool, []float32) {
	var Log string = ""
	var cost []float64
	var count = len(myNode.RefPoints)

	if is_sck { // If it is self-crosscheck, set d0 = 0
		cost = append(cost, 0)
	}

	for i := 0; i < count; i++ { // get distance(= delay) vector between this node and reference points
		rtt, err := GetAvgRTT(myNode.RefPoints[i].NodeIp)
		if int(myNode.MyId) == 0 {
			var rtt_changed [7]float64 = [7]float64{32.0, 18.0, 10.0, 40.0, 38.0, 60.0, 0.5}
			rtt = rtt_changed[i]
		}
		if err != nil {
			log.Println(err)
			return nil, errors.New("[error] Error generated while calculating ping -- Can't get avgRtt!"), Log, nil, nil
		}
		Log += fmt.Sprintf("delay(%d <---> %d) %f\n", int(myNode.RefPoints[i].Id), int(myNode.MyId), rtt)
		cost = append(cost, rtt)
	}
	return (get_node_ndim_coord(int(myNode.EventSig), myNode.RefPoints, int(myNode.Ndim), int(myNode.MyId), cost, Log)) // calculate coordinate of new node
}
