package fec

import (
	"sort"
	"time"

	"github.com/lucas-clemente/quic-go/internal/protocol"
	"github.com/lucas-clemente/quic-go/internal/wire"
)

type FecBuffer struct {
	firstReceived         bool
	recvComplete          bool
	symbolSize            uint32
	numSrcSymbols         uint32
	numEncSymbols         uint32
	recvSizeForDecode     uint32
	srcBlockSize          uint32
	codeRate              float32
	streamID              protocol.StreamID
	startStreamOffset     protocol.ByteCount
	endStreamOffset       protocol.ByteCount
	nextFrameOffset       protocol.ByteCount // In order
	streamFrameDataLength protocol.ByteCount
	fecFrameDataLength    protocol.ByteCount
	oobStreamFrameQueue   []wire.StreamFrame
	oobFecFrameQueue      []wire.FecFrame
	encBlock              []byte
	bufferMap             []uint32 // unit of symbols
	timer                 *time.Timer
	startTime             time.Time
	timerStop             chan bool
	fecHandler            *FecHandler
	decodeStarted         bool
	packetNumberMap       map[protocol.ByteCount]protocol.PacketNumber
}

func (b *FecBuffer) Init(streamID protocol.StreamID, startOffset protocol.ByteCount, symSize uint32, numSrcSyms uint32, fhandler *FecHandler) {
	b.firstReceived = true
	b.recvComplete = false
	b.symbolSize = symSize
	b.numSrcSymbols = numSrcSyms
	b.codeRate = 0.8
	b.numEncSymbols = uint32(float32(numSrcSyms) / b.codeRate)
	b.recvSizeForDecode = uint32(b.symbolSize * 18) //155 when k=128, 18 when k=16
	b.srcBlockSize = symSize * numSrcSyms
	b.streamID = streamID
	b.startStreamOffset = startOffset
	b.endStreamOffset = startOffset + protocol.ByteCount(b.srcBlockSize) - 1
	b.nextFrameOffset = startOffset
	b.streamFrameDataLength = 0
	b.fecFrameDataLength = 0
	b.encBlock = make([]byte, b.symbolSize*b.numEncSymbols)
	b.bufferMap = make([]uint32, b.numEncSymbols)
	for i, _ := range b.bufferMap {
		b.bufferMap[i] = 0
	}

	b.startTime = time.Now()
	b.fecHandler = fhandler
	b.oobStreamFrameQueue = make([]wire.StreamFrame, 0)
	b.oobFecFrameQueue = make([]wire.FecFrame, 0)
	b.decodeStarted = false
	b.timerStop = make(chan bool)

	// Start timer
	b.timer = time.NewTimer(time.Millisecond * 200) // Timout for Raptor decoding (ms)
	go func() {
		select {
		case <-b.timer.C:
			b.fecHandler.timeoutChan <- b
			b.decodeStarted = true
		case <-b.timerStop:
			FecLog("fec_buffer.go: Destroy(): Timer Stop (Block=%d-%d)", b.startStreamOffset, b.endStreamOffset)
		}
	}()

	b.packetNumberMap = make(map[protocol.ByteCount]protocol.PacketNumber)
}

func (b *FecBuffer) PushStreamFrame(streamFrame *wire.StreamFrame, packetNumber protocol.PacketNumber) {
	// Set position in the FEC buffer
	startOffset := streamFrame.Offset
	endOffset := startOffset + streamFrame.DataLen() - 1

	// Check duplicated frame
	if startOffset < b.nextFrameOffset && endOffset < b.nextFrameOffset {
		// Fully duplicated
		FecLog("fec_buffer.go: PushStreamFrame(): Stream Frame is fully duplicated! (Block=%d-%d)(Frame offset=%d-%d)(nextFrameOffset=%d) \n",
			b.startStreamOffset, b.endStreamOffset, startOffset, endOffset, b.nextFrameOffset)
		return
	}

	// TODO: Do we need to consider partially duplicated cases?

	// Append frame into queue
	prevNextFrameOffset := b.nextFrameOffset
	if startOffset == b.nextFrameOffset {
		// if ther is no gap between nextFrameOffset and startOffset then copy frame data to encBlock
		startPos := streamFrame.Offset - b.startStreamOffset
		endPos := startPos + protocol.ByteCount(streamFrame.DataLen())
		copy(b.encBlock[startPos:endPos], streamFrame.Data)
		// FecLog("fec_handler.go: PushStreamFrame(): Push Stream Frame (Block=%d-%d)(Frame offset=%d-%d)(nextFrameOffset=%d)(oobStreamFrameQueue=%d)",
		// 	b.startStreamOffset, b.endStreamOffset, startOffset, endOffset, b.nextFrameOffset, len(b.oobStreamFrameQueue))

		// Update nextFrameOffset
		b.nextFrameOffset = endOffset + 1

		// Update nextFrameOffset if out-of-order queue is not empty
		for i, frame := range b.oobStreamFrameQueue {
			if frame.Offset == b.nextFrameOffset {
				b.nextFrameOffset = frame.Offset + frame.DataLen()
				// FecLog("fec_handler.go: PushStreamFrame(): Update nextFrameOffset (Block=%d-%d)(Frame offset=%d-%d)(nextFrameOffset=%d)(oobStreamFrameQueue=%d)",
				// 	b.startStreamOffset, b.endStreamOffset, frame.Offset, frame.Offset+protocol.ByteCount(frame.DataLen())-1,
				// 	b.nextFrameOffset, len(b.oobStreamFrameQueue))
			} else {
				b.oobStreamFrameQueue = b.oobStreamFrameQueue[i:]
				break
			}
		}
	} else if startOffset > b.nextFrameOffset {
		// If there exists gap between nextFrameOffset and startOffset then insert frame into out-of-order queue
		b.oobStreamFrameQueue = append(b.oobStreamFrameQueue, *streamFrame)
		// FecLog("fec_handler.go: PushStreamFrame(): Insert into OOB Queue (Block=%d-%d)(Frame offset=%d-%d)(nextFrameOffset=%d)(oobStreamFrameQueue=%d)",
		// 	b.startStreamOffset, b.endStreamOffset, streamFrame.Offset, streamFrame.Offset+protocol.ByteCount(streamFrame.DataLen())-1,
		// 	b.nextFrameOffset, len(b.oobStreamFrameQueue))

		if len(b.oobStreamFrameQueue) > 1 {
			sort.Slice(b.oobStreamFrameQueue, func(i, j int) bool {
				return (b.oobStreamFrameQueue[i].Offset < b.oobStreamFrameQueue[j].Offset)
			})
		}
	}

	b.streamFrameDataLength += streamFrame.DataLen()

	if streamFrame.DataLen() != 1024 {
		FecLog("fec_buffer.go: PushStreamFrame(): streamFrame.DataLen() != 1024 (StreamFrame Offset=%d)(Len=%d)",
			streamFrame.Offset, streamFrame.DataLen())
	}

	// Set buffer map
	b.setBufferMap(prevNextFrameOffset, b.nextFrameOffset)

	// Append packet number for the stream frame
	b.packetNumberMap[streamFrame.Offset] = packetNumber
	//FecLog("fec_buffer.go: PushStreamFrame(): frame offset=%d, packet number=%d", streamFrame.Offset, packetNumber)
}

func (b *FecBuffer) PushFecFrame(fecFrame *wire.FecFrame) {
	// Set position in the FEC buffer
	startOffset := b.endStreamOffset + 1 + fecFrame.FecDataOffset
	endOffset := startOffset + fecFrame.FecDataLen() - 1

	// Check duplicated frame
	if startOffset < b.nextFrameOffset && endOffset < b.nextFrameOffset {
		// Fully duplicated
		FecLog("fec_handler.go: PushFecFrame(): FEC Frame is fully duplicated! (Block=%d-%d)(Frame offset=%d-%d)(nextFrameOffset=%d) \n",
			b.startStreamOffset, b.endStreamOffset, startOffset, endOffset, b.nextFrameOffset)
		return
	}

	// TODO: Do we need to consider partially duplicated cases?

	// Append frame into queue
	prevnextFrameOffset := b.nextFrameOffset
	if startOffset == b.nextFrameOffset {
		// if ther is no gap between nextFrameOffset and startOffset then copy frame data to encBlock
		startPos := startOffset - b.startStreamOffset
		endPos := startPos + protocol.ByteCount(fecFrame.FecDataLen())
		FecLog("fec_handler.go: PushFecFrame(): Push FEC Frame (Block=%d-%d)(Frame offset=%d-%d)(nextFrameOffset=%d)(oobFecFrameQueue=%d)",
			b.startStreamOffset, b.endStreamOffset, startOffset, endOffset, b.nextFrameOffset, len(b.oobFecFrameQueue))

		copy(b.encBlock[startPos:endPos], fecFrame.FecData)

		// Update nextFrameOffset
		b.nextFrameOffset = endOffset + 1

		// Update nextFrameOffset if out-of-order queue is not empty
		for i, frame := range b.oobFecFrameQueue {
			offset := b.endStreamOffset + 1 + frame.FecDataOffset
			if offset == b.nextFrameOffset {
				b.nextFrameOffset = offset + frame.FecDataLen()
				// FecLog("fec_handler.go: PushFecFrame(): Update nextFrameOffset (Block=%d-%d)(Frame offset=%d-%d)(nextFrameOffset=%d)(oobFecFrameQueue=%d)",
				// 	b.startStreamOffset, b.endStreamOffset, frame.FecDataOffset, frame.FecDataOffset+protocol.ByteCount(frame.FecDataLen())-1,
				// 	b.nextFrameOffset, len(b.oobFecFrameQueue))
			} else {
				b.oobFecFrameQueue = b.oobFecFrameQueue[i:]
				break
			}
		}
	} else if startOffset > b.nextFrameOffset {
		// If there exists gap between nextFrameOffset and startOffset then insert frame into out-of-order queue
		b.oobFecFrameQueue = append(b.oobFecFrameQueue, *fecFrame)
		FecLog("fec_handler.go: PushFecFrame(): Insert into OOB Queue (Block=%d-%d)(Frame offset=%d-%d)(nextFrameOffset=%d)(oobFecFrameQueue=%d)",
			b.startStreamOffset, b.endStreamOffset, fecFrame.FecDataOffset, fecFrame.FecDataOffset+protocol.ByteCount(fecFrame.FecDataLen())-1,
			b.nextFrameOffset, len(b.oobFecFrameQueue))

		if len(b.oobFecFrameQueue) > 1 {
			sort.Slice(b.oobFecFrameQueue, func(i, j int) bool {
				return (b.oobFecFrameQueue[i].FecDataOffset < b.oobFecFrameQueue[j].FecDataOffset)
			})
		}
	}

	b.fecFrameDataLength += fecFrame.FecDataLen()

	if fecFrame.FecDataLen() != 1024 {
		FecLog("fec_buffer.go: PushFecFrame(): fecFrame.FecDataLen() != 1024 (FecFrame Offset=%d)(Len=%d)",
			fecFrame.FecDataOffset, fecFrame.FecDataLen())
	}

	// Set buffer map
	b.setBufferMap(prevnextFrameOffset, b.nextFrameOffset)
}

func (b *FecBuffer) setBufferMap(startOffset protocol.ByteCount, endOffset protocol.ByteCount) {
	startSymIndex := uint32(startOffset-b.startStreamOffset) / uint32(b.symbolSize)
	endSymIndex := uint32(endOffset-b.startStreamOffset) / uint32(b.symbolSize)
	remainingLength := endOffset - startOffset
	// FecLog("fec_buffer.go: setBufferMap(): Buffer=%d, offset=%d-%d, symbol=%d-%d (b.bufferMap len=%d, remainingLength=%d)",
	// 	b.startStreamOffset, startOffset, endOffset, startSymIndex, endSymIndex, len(b.bufferMap), remainingLength)

	if int(endSymIndex) == len(b.bufferMap) {
		endSymIndex -= 1
	}

	for i := startSymIndex; i <= endSymIndex; i++ {
		diff := b.symbolSize - b.bufferMap[i]
		if remainingLength >= protocol.ByteCount(diff) {
			b.bufferMap[i] = b.bufferMap[i] + diff
			remainingLength = remainingLength - protocol.ByteCount(diff)
		} else {
			b.bufferMap[i] = b.bufferMap[i] + uint32(remainingLength)
			remainingLength = 0
			break
		}
		// FecLog("fecBuffer.go: setBufferMap(): remainingLength=%d", remainingLength)
	}
}

func (b *FecBuffer) GetEncodeBlock() ([]byte, []uint32, int) {
	// For Stream frames
	for _, streamFrame := range b.oobStreamFrameQueue {
		startPos := streamFrame.Offset - b.startStreamOffset
		endPos := startPos + protocol.ByteCount(streamFrame.DataLen())

		if streamFrame.DataLen() != 1024 {
			FecLog("streamFrame.Datalen() < 1024 2222222222 %d ", streamFrame.DataLen())
		}

		// Copy stream frame data
		copy(b.encBlock[startPos:endPos], streamFrame.Data)

		// Set symbol map
		startOffset := streamFrame.Offset
		endOffset := startOffset + protocol.ByteCount(streamFrame.DataLen())
		b.setBufferMap(startOffset, endOffset)
	}

	// For FEC frames
	for _, fecFrame := range b.oobFecFrameQueue {
		startPos := b.endStreamOffset + 1 + fecFrame.FecDataOffset - b.startStreamOffset
		endPos := startPos + protocol.ByteCount(fecFrame.FecDataLen())

		if fecFrame.FecDataLen() != 1024 {
			FecLog("fecFrame.FecDataLen() < 1024 2222222222 %d ", fecFrame.FecDataLen())
		}

		// Copy FEC frame data
		copy(b.encBlock[startPos:endPos], fecFrame.FecData)

		// Set symbol map
		startOffset := b.endStreamOffset + 1 + fecFrame.FecDataOffset
		endOffset := startOffset + protocol.ByteCount(fecFrame.FecDataLen())
		b.setBufferMap(startOffset, endOffset)
	}

	idx := 0
	lost := 0
	symbolMap := make([]uint32, b.numEncSymbols)
	encBlock2 := make([]byte, b.symbolSize*b.numEncSymbols)
	for i := 0; i < len(b.bufferMap); i++ {
		if b.bufferMap[i] == b.symbolSize {
			symbolMap[idx] = uint32(i)
			startPos1 := idx * int(b.symbolSize)
			endPos1 := startPos1 + int(b.symbolSize)
			startPos2 := i * int(b.symbolSize)
			endPos2 := startPos2 + int(b.symbolSize)
			copy(encBlock2[startPos1:endPos1], b.encBlock[startPos2:endPos2])
			idx++
		} else {
			lost++
			// FecLog("fec_buffer.go: GetEncodeBlock(): (Block Offset=%d)(bufferMap[%d]=%d)(lost=%d)",
			// 	b.startStreamOffset, i, b.bufferMap[i], lost)
		}
	}

	return encBlock2, symbolMap, lost
}

func (b *FecBuffer) Length() protocol.ByteCount {
	return b.streamFrameDataLength + b.fecFrameDataLength
}

func (b *FecBuffer) CheckDecodingPossible(fin bool) {
	if !b.decodeStarted {
		if b.streamFrameDataLength >= protocol.ByteCount(b.srcBlockSize) {
			// If all of stream frames are received without loss, start decoding
			b.fecHandler.streamFinChan <- b
			b.decodeStarted = true
		} else if b.streamFrameDataLength+b.fecFrameDataLength >= protocol.ByteCount(b.recvSizeForDecode) {
			// If number received symbols are enough to decode, start decoding
			b.fecHandler.enoughRecvChan <- b //(b.streamFrameDataLength + b.fecFrameDataLength)
			b.decodeStarted = true
		} else if fin {
			b.fecHandler.finalChan <- b
			b.decodeStarted = true
		}
	}
}

func (b *FecBuffer) Destroy() {
	// b.streamFrameQueue = nil
	// b.fecFrameQueue = nil
	// b.oobStreamFrameQueue = nil
	// b.oobFecFrameQueue = nil
	// b.encBlock = nil
	// b.bufferMap = nil
	b.timerStop <- true
	b.timer.Stop()
}
