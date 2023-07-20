package utils

/* THIS FILE SAVES CORE DATA AS A GOBAL VARIABLE */

const NodeTotal int = 80 // 60 // Total number of peer nodes

var Ndim int = 3             // dimension
var Nb_Ref int = 7           // the number of reference point when getting peer's coordinate
var Nb_SCK_Ref int = Ndim    // the number of reference point for sck
var Origin_tresh float32 = 3 // when sck delay is over this value, we will restart implementing coordinate system
var NodeCount int = 0        // plus it when we add new node on coordinate system, while validating the nodes

const SCKInterval int = 5

var NodeData []NodeInfo   // Save id + coordinate + ip of peer nodes here
var TestData [][2]float32 // Test data (estimated delay vs test delay)

var IPTABLE [NodeTotal]string = [NodeTotal]string{} // ip address of peers

var PEER_ERROR int = 0    // count the number of re-request from peers
var RESTART_COUNT int = 0 // the number of restart

const OmtreeIP string = "172.17.0.4" //"172.19.0.99"
const OmtreePortNum string = "12345"
