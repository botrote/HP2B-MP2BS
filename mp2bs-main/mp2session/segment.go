package mp2session

import (
	"math"
	config "mp2bs/config"
	util "mp2bs/util"
	"sync"
)

type Segment struct {
	mutex              sync.Mutex
	BlockNumber        uint32
	SessionID          uint32
	Index              uint32
	startDataNumber    uint32
	endDataNumber      uint32
	expectedDataNumber uint32
	readDataOffset     uint32
	lastRecvDataOffset uint32 // inorder
	size               uint32
	Complete           bool
	Last               bool
	recvBuffer         []byte
	recvDataSize       []uint32

	// FEC MODE
	symSize            uint32
	numSrcSymbols      uint32
	numEncSymbols      uint32
	numDataNumber      uint32
	numRedDataNumber   uint32
	startRedDataNumber uint32
	endRedDataNumber   uint32
	recvRedBuffer      []byte
	recvRedDataSize    []uint32
	FecFinish          bool
}

func CreateSegment(blockNumber uint32, sessionID uint32, idx uint32,
	start uint32, end uint32, segmentSize uint32, last bool) (*Segment, error) {
	g := Segment{
		BlockNumber:        blockNumber,
		SessionID:          sessionID,
		Index:              idx,
		startDataNumber:    start,
		endDataNumber:      end,
		size:               segmentSize,
		expectedDataNumber: start,
		readDataOffset:     0,
		lastRecvDataOffset: 0,
		Complete:           false,
		Last:               last,
		numDataNumber:      0,
		numRedDataNumber:   0,
		recvBuffer:         make([]byte, segmentSize),
		recvDataSize:       make([]uint32, end-start+1),
	}

	numData := int(end - start + 1)
	// TODO: No need initialization?
	for i := 0; i < numData; i++ {
		g.recvDataSize[i] = 0
	}

	util.Log("Segment.CreateSegment(): %d-%d (Size=%d)", g.startDataNumber, g.endDataNumber, g.size)

	return &g, nil
}

func CreateSegmentWithRedundant(blockNumber uint32, sessionID uint32, idx uint32,
	start uint32, end uint32, segmentSize uint32, last bool,
	symSize uint32, numSrcSym uint32, numEncSym uint32) (*Segment, error) {

	g, _ := CreateSegment(blockNumber, sessionID, idx, start, end, segmentSize, last)

	g.symSize = symSize
	g.numSrcSymbols = numSrcSym
	g.numEncSymbols = numEncSym
	redundantSize := g.symSize * (g.numEncSymbols - g.numSrcSymbols)

	g.startRedDataNumber = end + 1
	g.endRedDataNumber = g.startRedDataNumber + uint32(math.Ceil(float64(redundantSize)/float64(config.PAYLOAD_SIZE))) - 1
	g.recvRedBuffer = make([]byte, redundantSize)
	g.recvRedDataSize = make([]uint32, g.endRedDataNumber-g.startRedDataNumber+1)
	g.FecFinish = false

	numData := int(g.endRedDataNumber - g.startRedDataNumber + 1)
	// TODO: No need initialization?
	for i := 0; i < numData; i++ {
		g.recvRedDataSize[i] = 0
	}

	util.Log("Segment.CreateSegmentWithRedundant(): blockNumber=%d, %d-%d(Redundant:%d-%d, Size=%d)",
		g.BlockNumber, g.startDataNumber, g.endDataNumber, g.startRedDataNumber, g.endRedDataNumber, redundantSize)

	return g, nil
}

func (g *Segment) PushData(blockNumber uint32, dataNumber uint32, data []byte) bool {
	ret := false

	g.mutex.Lock()

	if blockNumber == g.BlockNumber && dataNumber >= g.startDataNumber && dataNumber <= g.endDataNumber {
		dataIdx := dataNumber - g.startDataNumber // data number
		startIdx := dataIdx * config.PAYLOAD_SIZE // bytes
		endIdx := startIdx + uint32(len(data))    // bytes
		copy(g.recvBuffer[startIdx:endIdx], data)
		g.recvDataSize[dataIdx] = uint32(len(data))
		g.numDataNumber++

		// Update expectedDataNumber and lastRecvDataOffset
		if dataNumber == g.expectedDataNumber {
			for g.recvDataSize[dataIdx] > 0 {
				if g.expectedDataNumber == g.endDataNumber {
					g.Complete = true
				}
				g.lastRecvDataOffset += g.recvDataSize[dataIdx]
				g.expectedDataNumber++
				dataIdx = g.expectedDataNumber - g.startDataNumber
				if dataIdx == uint32(len(g.recvDataSize)) {
					break
				}
			}
		}

		ret = true
	}

	g.mutex.Unlock()

	return ret
}

// For FEC Mode
func (g *Segment) PushRedData(blockNumber uint32, dataNumber uint32, data []byte) bool {
	ret := false

	g.mutex.Lock()

	if blockNumber == g.BlockNumber && dataNumber >= g.startRedDataNumber && dataNumber <= g.endRedDataNumber {
		dataIdx := dataNumber - g.startRedDataNumber // data number
		startIdx := dataIdx * config.PAYLOAD_SIZE    // bytes
		endIdx := startIdx + uint32(len(data))       // bytes
		copy(g.recvRedBuffer[startIdx:endIdx], data)
		g.recvRedDataSize[dataIdx] = uint32(len(data))
		g.numRedDataNumber++

		ret = true
	}

	g.mutex.Unlock()

	return ret
}

func (g *Segment) PopData(buf []byte) (int, bool) {
	g.mutex.Lock()

	readLen := uint32(0)
	bufLen := uint32(len(buf))

	util.Log("Segment.PopData(): readDataOffset=%d, lastRecvDataOffset=%d, size=%d ", g.readDataOffset, g.lastRecvDataOffset, g.size)

	// RecvBuffer is not empty
	if g.readDataOffset < g.lastRecvDataOffset {
		unreadDataSize := g.lastRecvDataOffset - g.readDataOffset
		if bufLen < unreadDataSize {
			// Length of buf is less than unread data size
			readLen = bufLen
		} else {
			// Length of buf is greater than that of readBuffer
			readLen = unreadDataSize
		}

		copy(buf, g.recvBuffer[g.readDataOffset:g.readDataOffset+readLen])
		g.readDataOffset += readLen
	}

	g.mutex.Unlock()

	return int(readLen), (g.readDataOffset == g.size)
}

func (g *Segment) hasDataToRead() bool {
	return (g.readDataOffset < g.lastRecvDataOffset)
}

// TODO
// func (g *Segment) FecModePopData(buf []byte) (int, bool) {
// 	g.mutex.Lock()
// 	readLen := uint32(0)
// 	bufLen := uint32(len(buf))

// 	multipath.Log("Segment.PopData(): readDataOffset=%d, lastRecvDataOffset=%d, size=%d ", g.readDataOffset, g.lastRecvDataOffset, g.size)
// 	if g.readDataOffset < g.lastRecvDataOffset { // recvBuffer is not empty
// 		unreadDataSize := g.lastRecvDataOffset - g.readDataOffset
// 		if bufLen < unreadDataSize {
// 			// length of buf is less than unread data size
// 			readLen = bufLen
// 		} else {
// 			// length of buf is greater than that of readBuffer
// 			readLen = unreadDataSize
// 		}

// 		copy(buf, g.recvBuffer[g.readDataOffset:g.readDataOffset+readLen])
// 		g.readDataOffset += readLen
// 	}
// 	g.mutex.Unlock()

// 	return int(readLen), (g.readDataOffset == g.size)
// }

// TODO
func (g *Segment) FecDecode(raptor *Raptor) bool {
	if g.numDataNumber+g.numRedDataNumber >= raptor.NumSymbolsForDecode {

		// SymbolMap
		// symbolMap

		// Raptor decoding
		// decBlock := raptor.Decode(g.recvBuffer, symbolMap)

		// Copy to recvBuffer
		// copy(g.recvBuffer[0:SEGMENT_SIZE], decBlock)

		//g.Complete = true

		return true
	}

	return false
}
