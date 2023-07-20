package utils

import (
	"fmt"
)

/*
*** # VALIDATOR_UTILS_INIT.GO
***
****  This file mainly contains the functions that initialize settings
****  These functions can be used when first starting the validator for cooordinate sysetem,
****	or when we have to re-implement the coordinate system again.
***
 */

// ==================== INITIALIZE SETTINGS ==================

// RESET COORDINATE DATA, FOR RE-IMPLEMENTATION OF COORDINATE SYSTEM
func Reset_coordinate_data() {
	for i := 0; i < NodeTotal; i++ {
		NodeData[i].Coord = make([]float32, Ndim)
	}
	NodeCount = 0
}

// INITIALIZE DATABASE BY ALLOCATING MEMORY ON THE STORAGE
func Initialize_database() {
	// allocate dynamic array to save coordinate data.
	// allocate arbitary ipaddress for this test
	for i := 0; i < NodeTotal; i++ {
		IPTABLE[i] = "172.19.0." + fmt.Sprintf("%d", i+100) + ":1053"
		//fmt.Println(IPTABLE[i])

	}
	TestData = make([][2]float32, NodeTotal)
	NodeData = make([]NodeInfo, NodeTotal)
	for i := 0; i < NodeTotal; i++ {
		NodeData[i].NodeIp = IPTABLE[i]
		NodeData[i].Id = int32(i)
		NodeData[i].Coord = make([]float32, Ndim)
	}
}
