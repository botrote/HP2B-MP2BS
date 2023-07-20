package utils

import (
	"errors"
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

/*
*** # PEER_UTILS_NDIM_COORD.GO
**** this file mainly contains the arithmetic functions.
**** and these functions use sort of equations to calculate new coordinate of this peer node.
 */

/*
*** # POSSIBLE ISSUES
*** 1. error-handling : i think cost[i] can't be zero. shall i have to handle error from that?
***  (ex. when it comes to pow2 cost[i], should i have to do error handling such as 'if (cost[i]!=0)' ??)
***
***	2. nan value : it will have nan-value if execute sqrt on a negative value
***  (if so, we change the signs of ref-nodes before recalculating the coord. if the value doesn't exists even after all the process, this peer node will request new refPoints to a validator)
***
*** 3. matrix non-inverstable : when getting coordinate of Node[K]
***   if it's K > dimension, it processes matrix operation to find it's coordinate. so we have to request new refpoints or more refpoints in this situation.
 */

// GET COORDINATE OF THE NODE IN GIVEN DIMENSION
func get_node_ndim_coord(eventSig int, refPoints []*NodeInfo, ndim int, nodeId int, cost []float64, Log string) ([]float32, error, string, []bool, []float32) {
	sign := make([]bool, 0)
	if ndim < 0 || nodeId < 0 { // ** exception-handling
		return nil, errors.New("get_node_ndim_coord() : ndim or nodeId parameter cannot have negative value"), "", sign, nil
	}
	result := make([]float32, ndim) // coord of this node
	// ====================== IMPLEMENT THE AXIS-NODE ======================
	if eventSig == 1 && nodeId <= ndim {
		if nodeId < len(refPoints) {
			return nil, errors.New("get_node_ndim_coord() : not enough reference nodes given"), "", sign, nil
		}
		if nodeId > 1 {
			// make up the coordinate system (until there comes full-axis coordinate)
			return (implement_axis_node(refPoints, ndim, nodeId, cost, Log))
		} else if nodeId == 1 {
			result[0] = float32(cost[0])
			return result, nil, Log, sign, check_test_performance(result, cost[0])
		} else { // nodeId == 0
			test := make([]float32, 2)
			return result, nil, Log, sign, test
		}
	} else {
		// ====================== AFTER THERE IS FULL AXIS-NODE IMPLEMENTED ======================
		var err error
		// EQUATION : Ax = b  =>  x = (((A^T)*A)^-1)*(A^T)*b
		l := len(cost) // len - 1 -> the number of neighbors
		len := len(refPoints)
		term1 := mat.NewDense(ndim, ndim, nil) // (A^T)*A
		term2 := mat.NewDense(ndim, ndim, nil) // ((A^T)*A)^-1
		term3 := mat.NewDense(ndim, l-1, nil)  // term2 *(A^T)
		set := mat.NewDense(ndim, 1, nil)      // term3 * b
		m := mat.NewDense(l-1, ndim, nil)      // A
		b := mat.NewDense(l-1, 1, nil)         // [x, y, z]

		if len < ndim { // ** exception-handling
			return nil, errors.New("get_node_ndim_coord() : not enough reference nodes given"), "", sign, nil
		}
		var k int = 0
		if eventSig == 1 {
			k = 1
		} else if eventSig == 2 {
			m.SetRow(len-1, convertTo64(refPoints[len-1].Coord))
		}
		for i := 1; i < len; i++ {
			m.SetRow(i-1, convertTo64(refPoints[k].Coord)) // set coord matrix of refpoints
			tmp := -math.Pow(cost[i], 2)
			if cost[0] != 0 {
				tmp += math.Pow(cost[0], 2)
			}
			for j := 0; j < ndim; j++ {
				if refPoints[k].Coord[j] != 0 {
					tmp += math.Pow(float64(refPoints[k].Coord[j]), 2)
				}
			}
			r := []float64{tmp / 2}
			b.SetRow(i-1, r)
			k++
		}
		term1.Mul(m.T(), m)
		err = term2.Inverse(term1)
		if err != nil {
			return nil, errors.New("Error generated while inversing a matrix.\n"),
				Log + "\n A = \n" + fmt.Sprintln(mat.Formatted(m)) + "\n B = \n" + fmt.Sprintln(mat.Formatted(b)) +
					"\n[error] Matrix not invertable\n", sign, nil
		}
		term3.Mul(term2, m.T())
		set.Mul(term3, b)
		for i := 0; i < ndim; i++ {
			result[i] = float32(set.At(i, 0))
		}

		return result, nil,
			Log + "\n A = \n" + fmt.Sprintln(mat.Formatted(m)) + "\n B = \n" + fmt.Sprintln(mat.Formatted(b)) + "\n C = \n" + fmt.Sprintln(mat.Formatted(set)), sign, check_test_performance(result, cost[0])
	}
	return nil, errors.New("[error] get_node_ndim_coord() : unknown error!"), Log, sign, nil
}

// IMPLEMENT THE AXIS NODE AND DETERMINE THE SIGN OF THE LAST COORD VALUE OF REFPOINTS(= AXIS-NODES BEFORE THIS NODE)
func implement_axis_node(refPoints []*NodeInfo, ndim int, nodeId int, cost []float64, Log string) ([]float32, error, string, []bool, []float32) {

	var sum float64 = 0
	var tmp float64 = 0

	temp := make([]bool, 0) // empty array (= if there is no change in sign)
	arr := make([]bool, nodeId-1)
	sign := make_sign_comb(nodeId-1, arr, 0) // sign can be either + or -
	fmt.Println("sign comb", sign, "\n")

	// MAKE LOG STRING OF ORIGINAL COORDINATE
	var original_points string
	for i := 0; i < len(refPoints); i++ {
		original_points += fmt.Sprintln(refPoints[i].Coord)
	}
	// GET THE COORDINATE OF NODE[K] WITH APPLYING ALL POSSIBLE CASES OF SIGN COMBINATION
	for k := 0; k < len(sign); k++ {
		sum = 0
		tmp = 0
		result := make([]float32, ndim) // coord of this node
		// CHANGE SIGN OF THE COORDS
		fmt.Printf("\nsign array [%d] :", k)
		fmt.Println(sign[k])
		fmt.Println("\nChange sign of coord [before] :", original_points)
		for p := 0; p < len(sign[k]); p++ {
			if sign[k][p] == false {
				refPoints[p+1].Coord[p] *= -1 // set new coordinate signs
			}
		}
		fmt.Println("\nChange sign of coord [after] :")
		for i := 0; i < len(refPoints); i++ {
			fmt.Println(refPoints[i].Coord)
		}
		// GET (0 ~ K-1)TH COORD OF REFERENCE NODE[K]
		for i := 1; i < nodeId; i++ {
			for j := 0; j < i-1; j++ {
				sum += (math.Pow(float64(refPoints[i].Coord[j]), 2) - float64(2*refPoints[i].Coord[j]*result[j]))
			}
			sum += (math.Pow(cost[0], 2) - math.Pow(cost[i], 2))
			if refPoints[i].Coord[i-1] != float32(0) { // prevent 0 power 2 -> makes NaN values
				sum += math.Pow(float64(refPoints[i].Coord[i-1]), 2)
				sum = sum / (2 * float64(refPoints[i].Coord[i-1]))
			}
			result[i-1] = float32(sum)
		}
		// GET (K)TH COORD OF REFERENCE NODE[K]
		term := float64(0)
		for i := 0; i < nodeId-1; i++ {
			if float64(result[i]) != float64(0) {
				term -= math.Pow(float64(result[i]), 2)
			}
		}
		if cost[0] != 0 {
			term += math.Pow(cost[0], 2)
		}
		if term != float64(0) {
			result[nodeId-1] = float32(math.Sqrt(term))
		}
		// GO TO THE NEXT LOOP TO FIND NEW COORDINATE THAT IS APPLYING THE OTHER COMBINATION OF SIGN
		if math.IsNaN(float64(result[nodeId-1])) {
			// when NaN value is found (= when there is neg-value under sqrt)
			tmp = term
			fmt.Println("value under square root : ", tmp)
			for p := 0; p < len(sign[k]); p++ {
				if sign[k][p] == false {
					refPoints[p+1].Coord[p] *= -1 // recover original coordinate signs
				}
			}
		} else {
			fmt.Println("final sign : ", sign[k])
			if k == 0 {
				return result, nil, Log, temp, check_test_performance(result, cost[0])
			}
			return result, nil, Log, sign[k], check_test_performance(result, cost[0])
		}
	}
	return nil, errors.New(fmt.Sprintf("NaN value in (%d)th coord of node[%d] - there is neg-value under Sqrt", nodeId, nodeId)),
		Log + fmt.Sprintf("\n[error] Negative value under square root : %f\n", tmp), temp, nil // when we can't find value of this coord.
}

// =========================================================================================================================================

// MAKE ALL COMBINATION OF BOOL VECTORS (WE WILL USE 'FALSE' AS (-), 'TRUE' AS (+) AT THE CALLEE OF THIS FUNCTION)
func make_sign_comb(n int, arr []bool, i int) [][]bool {
	tmp := make([][]bool, 0)
	if i == n {
		res := make([][]bool, 1)
		res[0] = arr
		return res
	}
	newar := make([]bool, len(arr))
	copy(newar, arr)
	newar[i] = true
	a := make_sign_comb(n, newar, i+1)
	newar = make([]bool, len(arr))
	copy(newar, arr)
	newar[i] = false
	b := make_sign_comb(n, newar, i+1)

	tmp = append(tmp, a...)
	tmp = append(tmp, b...)
	return tmp

}

// CONVERT THE FORMAT OF VECTOR FROM FLOAT32 TO FLOAT64
func convertTo64(ar []float32) []float64 {
	newar := make([]float64, len(ar))
	var v float32
	var i int

	for i, v = range ar {
		newar[i] = float64(v)
	}
	return newar
}

// MAKE REPORT OF COORDINATE ACCURACY ('ESTIMATED DELAY BY COORDINATE', 'MESAURED DELAY')
func check_test_performance(dist []float32, delay float64) []float32 {
	var sum float64 = 0

	res := make([]float32, 2)
	for i := 0; i < len(dist); i++ {
		if dist[i] != 0 {
			sum += math.Pow(float64(dist[i]), 2)
		}
	}
	if sum != 0 {
		res[0] = float32(math.Sqrt(sum))
	}
	res[1] = float32(delay) // measured delay (ping)
	return res
}

// PRINT COORDINATE OF THE GIVEN REFPOINTS <- USE THIS FUNCTION IF YOU HAVE TO DO SOME DEBUG ON THIS PROGRAM
// func printCoords(refPoints []*NodeInfo) {
// 	var i int
//
// 	fmt.Println("--------- Coordinate ---------")
// 	for i = 0; i < len(refPoints); i++ {
// 		fmt.Printf("Node #%d --> ", refPoints[i].Id)
// 		fmt.Println(refPoints[i].Coord)
// 	}
// }
