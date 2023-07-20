package session

import (
	"mp2bs/util"
	"sync"
)

type RecvBuffer struct {
	mutex             sync.Mutex
	readSeqNumber     uint32
	recvSeqNumber     uint32
	expectedSeqNumber uint32
	readBuffer        []byte
	reorderBuffer     map[uint32]*DataPacket
}

func CreateRecvBuffer() *RecvBuffer {
	b := RecvBuffer{
		readSeqNumber:     0,
		recvSeqNumber:     0,
		expectedSeqNumber: 0,
		readBuffer:        make([]byte, 0),
		reorderBuffer:     make(map[uint32]*DataPacket),
	}

	return &b
}

// Push into readBuffer or reorderBuffer
func (b *RecvBuffer) PushPacket(packet *DataPacket) {
	b.mutex.Lock()

	util.Log("RecvBuffer.PushPacket(): PathID=%d, PacketSeq=%d, ExpectedSeq=%d, Len.readBuffer=%d, Len.reorderBuffer=%d",
		packet.PathID, packet.SeqNumber, b.expectedSeqNumber, len(b.readBuffer), len(b.reorderBuffer))

	if packet.SeqNumber == b.expectedSeqNumber {
		// If the received packet is in-order..
		// Push payload into readBuffer
		b.readBuffer = append(b.readBuffer, packet.Payload...)
		b.expectedSeqNumber++

		// Move all packets from reorderBuffer until detect the packet in out-of-order
		for {
			oooPacket, exists := b.reorderBuffer[b.expectedSeqNumber]

			if !exists {
				break
			}

			b.readBuffer = append(b.readBuffer, oooPacket.Payload...)
			delete(b.reorderBuffer, b.expectedSeqNumber)
			b.expectedSeqNumber++
		}
	} else {
		// If the received packet is out-of-order..
		// Insert the received Data Packet into reorderBuffer
		b.reorderBuffer[packet.SeqNumber] = packet
	}

	b.mutex.Unlock()
}

// Read from readBuffer
func (b *RecvBuffer) Read(buf []byte) int {
	b.mutex.Lock()

	readLen := 0
	bufLen := len(buf)
	readBufferLen := len(b.readBuffer)

	if readBufferLen > 0 {
		if readBufferLen < bufLen {
			// Length of readBuffer is less than that of buf
			readLen = readBufferLen

		} else {
			// Length of buf is greater than that of readBuffer
			readLen = bufLen
		}

		copy(buf, b.readBuffer[:readLen])
		b.readBuffer = b.readBuffer[readLen:]
	}

	b.mutex.Unlock()

	return readLen
}

func (b *RecvBuffer) IsEmpty() bool {
	return (len(b.readBuffer) == 0)
}

func (b *RecvBuffer) GetLength() int {
	return len(b.readBuffer)
}
