package fec

import (
	"fmt"
	"sync"
	"time"

	"github.com/lucas-clemente/quic-go/internal/protocol"
	"github.com/lucas-clemente/quic-go/internal/wire"
)

var FecEnable bool

type FecHandler struct {
	mutex             sync.Mutex
	symbolSize        uint32
	numSrcSymbols     uint32
	srcBlockSize      uint32
	maxBlockSize      uint32
	fecDataLength     uint32
	encQueue          []byte
	transmissionQueue []*wire.FecFrame
	decQueue          map[protocol.ByteCount]*FecBuffer
	alreadyDecQueue   map[protocol.ByteCount]FecBuffer
	startOffset       protocol.ByteCount
	endOffset         protocol.ByteCount
	nextBlockOffset   protocol.ByteCount
	raptor            Raptor
	recvBytes         uint32
	startRecv         time.Time
	decStarted        bool
	streamFinChan     chan *FecBuffer        // streamFinChan is used to notify the decoding routine that all stream frame is received without loss
	enoughRecvChan    chan *FecBuffer        // fecFinChan is used to notify the decoding routine that enought frames are received for decoding
	timeoutChan       chan *FecBuffer        // timeoutChan is used to notify the decoding routine that decoding should start when the waiting timer is expired
	finalChan         chan *FecBuffer        // finalChan is used to notify the Fin is received
	FecChan           chan *wire.StreamFrame // fecChan is connected with FecChan in receiveStream
	FecChanFinal      chan bool

	// TODO
	// packet size??
}

// Constructor
func NewFecHandler() *FecHandler {
	f := FecHandler{}
	f.symbolSize = 1024
	f.numSrcSymbols = 16 //128
	f.srcBlockSize = f.symbolSize * f.numSrcSymbols
	f.maxBlockSize = f.srcBlockSize + uint32(protocol.MaxPacketBufferSize) // with margin
	f.fecDataLength = f.symbolSize
	f.encQueue = make([]byte, 0, f.maxBlockSize)
	f.transmissionQueue = make([]*wire.FecFrame, 0)
	f.decQueue = make(map[protocol.ByteCount]*FecBuffer, 0)
	f.alreadyDecQueue = make(map[protocol.ByteCount]FecBuffer, 0)
	f.startOffset = 0
	f.endOffset = 0
	f.nextBlockOffset = 0
	f.raptor.RaptorInit()
	f.startRecv = time.Now()
	f.decStarted = false

	f.streamFinChan = make(chan *FecBuffer, 50)
	f.enoughRecvChan = make(chan *FecBuffer, 50)
	f.timeoutChan = make(chan *FecBuffer, 50)
	f.FecChan = make(chan *wire.StreamFrame, 50)
	f.finalChan = make(chan *FecBuffer, 1)
	f.FecChanFinal = make(chan bool, 1)

	// Start decoder
	go f.decode()

	return &f
}

// Encoding
func (f *FecHandler) PushToEncode(frame *wire.StreamFrame) {
	// Set start offset
	if len(f.encQueue) == 0 {
		f.startOffset = frame.Offset
	} else {
		f.startOffset = f.endOffset
	}

	// Push stream frame data into encQueue
	f.encQueue = append(f.encQueue, frame.Data...)
	//fmt.Printf("yunmin - fec_handler.go: PushToEncode(): Offset=%d, After push encQueue=%d\n", frame.Offset, len(f.encQueue))

	// Check queue size
	if len(f.encQueue) >= int(f.srcBlockSize) {
		// Copy to source block for Raptor encoding
		srcBlock := f.encQueue[0:f.srcBlockSize]

		// Raptor encoding
		fecBlock := f.raptor.RaptorEncode(srcBlock, f.symbolSize, f.numSrcSymbols)
		if fecBlock == nil {
			panic(fmt.Sprintf("yunmin - fec_handler.go: PushToEncode():Raptor Encode Fail! (Block=%d-%d)", f.startOffset, f.endOffset))
		}

		// Shift
		f.encQueue = f.encQueue[f.srcBlockSize:]
		f.endOffset = frame.Offset + frame.DataLen() // frame is the last of source block

		FecLog("fec_handler.go: PushToEncode(): FEC Encoding (Block=%d-%d)(fecBlockSize=%d)", f.startOffset, f.endOffset, len(fecBlock))

		// Push encoded data into transmission queue
		f.pushToTransmissionQueue(fecBlock, frame.StreamID)
	}
}

func (f *FecHandler) pushToTransmissionQueue(fecBlock []byte, streamID protocol.StreamID) {
	var totalLength uint32
	var fecDataOffset protocol.ByteCount

	totalLength = uint32(len(fecBlock))
	fecDataOffset = 0

	for {
		// Make a new FEC freame
		frame := &wire.FecFrame{FecData: make([]byte, f.fecDataLength)}
		frame.StreamID = streamID
		frame.StreamOffset = f.startOffset
		frame.SymbolSize = f.symbolSize
		frame.NumOfSymbols = f.numSrcSymbols
		frame.FecDataOffset = fecDataOffset
		frame.FecDataLength = f.fecDataLength
		frame.FecTotalLength = totalLength
		frame.FecData = fecBlock[:f.fecDataLength]

		// Shift
		fecBlock = fecBlock[f.fecDataLength:]
		if len(fecBlock) == 0 {
			frame.Fin = true
		} else {
			frame.Fin = false
		}

		f.transmissionQueue = append(f.transmissionQueue, frame)
		fecDataOffset += protocol.ByteCount(f.fecDataLength)

		if len(fecBlock) == 0 {
			break
		}
	}

	FecLog("fec_handler.go: pushToTransmissionQueue(): (Push size=%d)(transmissionQueue=%d)", totalLength, len(f.transmissionQueue))
}

func (f *FecHandler) PopFecFrame(maxBytes protocol.ByteCount) *wire.FecFrame {
	if len(f.transmissionQueue) == 0 {
		return nil
	}

	fecFrame := f.transmissionQueue[0]
	if protocol.ByteCount(fecFrame.SymbolSize) <= maxBytes {
		if len(f.transmissionQueue) > 1 {
			f.transmissionQueue = f.transmissionQueue[1:]
		} else {
			f.transmissionQueue = nil
		}
		return fecFrame
	} else {
		FecLog("fec_handler.go: PopFecFrame(): fecFrame.SymbolSize(%d) > maxBytes(%d)", fecFrame.SymbolSize, maxBytes)
		return nil
	}
}

func (f *FecHandler) GetFecTransmissionQueueLen() int {
	return len(f.transmissionQueue)
}

// Decoding
func (f *FecHandler) PushStreamFrameToDecode(streamFrame *wire.StreamFrame, packetNumber protocol.PacketNumber) {
	// Create or get buffer
	f.mutex.Lock()
	fecBuffer := f.getOrCreateFecBuffer(streamFrame.StreamID, streamFrame.Offset, true)
	if fecBuffer == nil {
		f.mutex.Unlock()
		return
	}

	// Push stream frame
	fecBuffer.PushStreamFrame(streamFrame, packetNumber)
	f.mutex.Unlock()

	if streamFrame.Fin {
		FecLog("fec_handler.go: PushStreamFrameToDecode(): Receive FIN!!!")
	}

	// Check whether decoding is possible
	fecBuffer.CheckDecodingPossible(streamFrame.Fin)
}

func (f *FecHandler) PushFecFrameToDecode(fecFrame *wire.FecFrame) {
	// Create or get Buffer
	f.mutex.Lock()
	fecBuffer := f.getOrCreateFecBuffer(fecFrame.StreamID, fecFrame.StreamOffset, false)
	if fecBuffer == nil {
		f.mutex.Unlock()
		return
	}

	// Push FEC frame
	fecBuffer.PushFecFrame(fecFrame)
	f.mutex.Unlock()

	// Check whether decoding is possible
	fecBuffer.CheckDecodingPossible(false)
}

// FEC Decode
func (f *FecHandler) decode() {
	for {
		var fecBuffer *FecBuffer
		var streamRecvComplete bool
		fin := false

		select {
		// Decoding Case #1: All of stream frames are received without loss
		case fecBuffer = <-f.streamFinChan:
			streamRecvComplete = true
			FecLog("fec_handler.go: decode(): All of stream frames are received without loss! (Block=%d-%d)(nextBlockOffset=%d)(recv size=%d)",
				fecBuffer.startStreamOffset, fecBuffer.endStreamOffset, f.nextBlockOffset, fecBuffer.Length())
		// Decoding Case #2: Enough frames are received for decoding
		case fecBuffer = <-f.enoughRecvChan:
			streamRecvComplete = false
			FecLog("fec_handler.go: decode(): Enough frames are received for decoding! (Block=%d-%d)(nextBlockOffset=%d)(recv size=%d)",
				fecBuffer.startStreamOffset, fecBuffer.endStreamOffset, f.nextBlockOffset, fecBuffer.Length())
		// Decoding Case #3: Timer is expired
		case fecBuffer = <-f.timeoutChan:
			streamRecvComplete = false
			timerElapsed := time.Since(fecBuffer.startTime)
			FecLog("fec_handler.go: decode(): Timer is expired(%dms)! (Block=%d-%d)(nextBlockOffset=%d)(recv size=%d/%d)(oob=%d/%d)",
				timerElapsed.Milliseconds(), fecBuffer.startStreamOffset, fecBuffer.endStreamOffset, f.nextBlockOffset,
				fecBuffer.streamFrameDataLength, fecBuffer.fecFrameDataLength, len(fecBuffer.oobStreamFrameQueue), len(fecBuffer.oobFecFrameQueue))
		// Final Channel is
		case fecBuffer = <-f.finalChan:
			streamRecvComplete = true
			fin = true
			f.FecChanFinal <- true // Make finish to pushFecDecodedFrame in receive_stream.go
		}

		f.mutex.Lock()
		// Update nextBlockOffset
		if fecBuffer.startStreamOffset == f.nextBlockOffset {
			f.nextBlockOffset = fecBuffer.endStreamOffset + 1

			// Update until find the not-yet received block
			for {
				buffer, exists := f.alreadyDecQueue[f.nextBlockOffset]
				if exists {
					delete(f.alreadyDecQueue, f.nextBlockOffset)
					f.nextBlockOffset = buffer.endStreamOffset + 1
				} else {
					break
				}
			}
		} else if fecBuffer.startStreamOffset > f.nextBlockOffset {
			// Create fecBuffer info and insert into alreadyDecQueue
			FecLog("fec_handler.go: decode(): Insert into alreadyDecQueue (Block=%d-%d)", fecBuffer.startStreamOffset, fecBuffer.endStreamOffset)
			decodedFecBuffer := FecBuffer{streamID: fecBuffer.streamID,
				startStreamOffset: fecBuffer.startStreamOffset, endStreamOffset: fecBuffer.endStreamOffset}
			f.alreadyDecQueue[decodedFecBuffer.startStreamOffset] = decodedFecBuffer
		} else {
			FecLog("fec_handler.go: decode(): startStreamOffset < nextBlockOffset! (Block=%d-%d)(nextBlockOffset=%d)",
				fecBuffer.startStreamOffset, fecBuffer.endStreamOffset, f.nextBlockOffset)
			//f.mutex.Unlock()
			panic("fecBuffer.startStreamOffset < f.nextBlockOffset")
		}

		// TODO: We have to consider the variable block size
		f.recvBytes += f.srcBlockSize

		// For Decoding Case #1, do not perform Raptor decoding
		// For Decoding Case #2 and #3, perform Raptor decoding
		if !streamRecvComplete {
			// Get encoding block (1-D byte array) and symbol map
			encBlock, symbolMap, lost := fecBuffer.GetEncodeBlock()

			// Perform Raptor decoding with loss information
			start := time.Now()
			decBlock := f.raptor.RaptorDecode(fecBuffer.startStreamOffset, encBlock, symbolMap, f.symbolSize, f.numSrcSymbols)
			elapsed := time.Since(start)

			if decBlock != nil {
				FecLog("fec_handler.go: decode(): Raptor decoding (Block=%d-%d)(Time=%dms)(DecBlockSize=%d)",
					fecBuffer.startStreamOffset, fecBuffer.endStreamOffset, elapsed.Milliseconds(), len(decBlock))

				// Push decoded data into frame queue
				f.pushReceiveStreamFrameQueue(decBlock, symbolMap, fecBuffer)

				if lost >= 1 && lost <= 3 {
					// 	FecLog("fec_handler.go: decode(): encBlock len=%d", len(encBlock)))
					// 	// FecBytesLog(encBlock, int(f.symbolSize*160)
					// 	fmt.Printf("\n\n\n-----------------------------------------------------------------------------------------------------------\n\n\n")
					// 	FecLog("fec_handler.go: decode(): decBlock len=%d", len(decBlock)))
					// 	// FecBytesLog(decBlock, int(f.symbolSize*f.numSrcSymbols)
				}
			} else {
				FecLog("fec_handler.go: decode(): Raptor Decode Fail!(Block=%d-%d)(Time=%dms)(DecBlockSize=%d)",
					fecBuffer.startStreamOffset, fecBuffer.endStreamOffset, elapsed.Milliseconds(), len(decBlock))
			}

			decBlock = nil
		}

		// Delete FEC buffer & decBlock
		delete(f.decQueue, fecBuffer.startStreamOffset)
		fecBuffer.Destroy()
		fecBuffer = nil

		elapsedRecv := time.Since(f.startRecv)
		FecLog("fec_handler.go: decode(): Receiving took %fs, Throughput=%fMbps (nextBlockOffset=%d)",
			elapsedRecv.Seconds(), float64(f.recvBytes)*8.0/elapsedRecv.Seconds()/(1024.0*1024.0), f.nextBlockOffset)

		f.mutex.Unlock()

		if fin {
			break
		}
	}
	FecLog("Fec_handler.go: decode(): Finish!")
}

func (f *FecHandler) pushReceiveStreamFrameQueue(decBlock []byte, symbolMap []uint32, fecBuffer *FecBuffer) {
	// Find lost symbols
	var prevPacketNumber protocol.PacketNumber
	prevIdx := int(0)
	nextIdx := uint32(0)
	for i := 0; i < int(f.numSrcSymbols); i++ {
		if symbolMap[i] != nextIdx {
			for j := i; j < int(symbolMap[i]) && j < int(f.numSrcSymbols); j++ {
				// Make a stream frame
				frame := &wire.StreamFrame{Data: make([]byte, f.symbolSize)}
				frame.StreamID = fecBuffer.streamID
				frame.Offset = fecBuffer.startStreamOffset + protocol.ByteCount(uint32(j)*f.symbolSize)
				startPos := uint32(j) * f.symbolSize
				endPos := startPos + f.symbolSize
				copy(frame.Data, decBlock[startPos:endPos])

				// Push to recovery queue
				f.FecChan <- frame

				// Add to packet number of recovered frame
				// Determine packet number of recovered frame
				pn := prevPacketNumber + protocol.PacketNumber(j-prevIdx)
				FecRecoveredPacketList = append(FecRecoveredPacketList, pn)

				FecLog("fec_handler.go: pushReceiveStreamFrameQueue(): Lost symbolMap[%d], packet number=%d, offset=%d, pos=%d-%d",
					j, pn, frame.Offset, startPos, endPos)
			}
		} else {
			offset := fecBuffer.startStreamOffset + protocol.ByteCount(uint32(i)*f.symbolSize)
			prevPacketNumber = fecBuffer.packetNumberMap[offset]
			prevIdx = i
		}
		nextIdx = symbolMap[i] + 1
	}
}

func (f *FecHandler) getOrCreateFecBuffer(streamID protocol.StreamID, offset protocol.ByteCount, flagStreamFrame bool) *FecBuffer {
	if offset >= f.nextBlockOffset {
		// Check decQueue
		for _, fecBuffer := range f.decQueue {
			if fecBuffer.streamID == streamID &&
				fecBuffer.startStreamOffset <= offset && offset <= fecBuffer.endStreamOffset {
				return fecBuffer
			}
		}

		// Check alreadyDecQueue (for out-of-order)
		for _, fecBuffer := range f.alreadyDecQueue {
			if fecBuffer.streamID == streamID &&
				fecBuffer.startStreamOffset <= offset && offset <= fecBuffer.endStreamOffset {
				return nil
			}
		}

		// if not exists, create a new buffer
		newFecBuffer := &FecBuffer{}
		num := offset / protocol.ByteCount(f.srcBlockSize)
		startOffset := num * protocol.ByteCount(f.srcBlockSize)
		newFecBuffer.Init(streamID, startOffset, f.symbolSize, f.numSrcSymbols, f)
		f.decQueue[startOffset] = newFecBuffer

		FecLog("fec_handler.go: getOrCreateFecBuffer(): Create FecBuffer (Block=%d-%d)(FrameOffset=%d)(nextBlockOffset=%d)(decQueueSize=%d)",
			newFecBuffer.startStreamOffset, newFecBuffer.endStreamOffset, offset, f.nextBlockOffset, len(f.decQueue))

		return newFecBuffer
	} else {
		// FecLog("fec_handler.go: getOrCreateFecBuffer(): Already Decoded! (FrameOffset=%d)(nextBlockOffset=%d)",
		// 	offset, f.nextBlockOffset)
		return nil
	}
}
