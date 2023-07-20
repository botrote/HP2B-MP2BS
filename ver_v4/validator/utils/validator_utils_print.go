package utils

import (
	"fmt"
)

/*
*** # VALIDATOR_UTILS_PRINT.GO
***
****  This file mainly contains the functions that print informative logs to check validating progress status
***
 */

// ==================== PRINT RESULT & LOGS ==================

func PrintTestResults() {
	for i := 0; i < NodeTotal; i++ {
		fmt.Printf("%f|%f|%f\n", TestData[i][0], TestData[i][1],
			TestData[i][0]-TestData[i][1]) // real | ideal | gap(error)
	}
}

func PrintRefpoints(in *TData) {
	fmt.Println("----------------------------------------")
	fmt.Printf(" reference points : ")
	for j := 0; j < len(in.RefPoints); j++ {
		fmt.Printf("%d,", in.RefPoints[j].Id)
	}
	fmt.Println("\n")
}

func PrintCoords() {
	var i int
	fmt.Println("--------- finalized Coordinate ---------")
	for i = 0; i < NodeTotal; i++ {
		if i == 0 || NodeData[i].Coord[0] != 0 {
			fmt.Printf("Node #%d --> ", i)
			for j := 0; j < len(NodeData[i].Coord); j++ {
				fmt.Printf(" %f", NodeData[i].Coord[j])
			}
			fmt.Println("")
		}
	}
	fmt.Println("----------------------------------------")
	// fmt.Printf("New Coordinate (#N%d) ", i-1)
	// fmt.Printf("New Coordinate (#N%d) ", i)
	// //fmt.Println(NodeData[i].Coord)
	// for j := 0; j < len(NodeData[i].Coord); j++ {
	// 	fmt.Printf(" %f", NodeData[i].Coord[j])
	// }
	// fmt.Println("")

}
