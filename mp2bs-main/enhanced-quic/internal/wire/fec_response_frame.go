package wire

import (
	"bytes"

	"github.com/lucas-clemente/quic-go/internal/protocol"
	"github.com/lucas-clemente/quic-go/quicvarint"
)

// A FEC Response Frame of QUIC
type FecRespFrame struct {
	StreamID       protocol.StreamID
	StreamOffset   protocol.ByteCount // Start of stream offset
	PacketLossRate uint32             // Packet loss rate
}

func parseFecRespFrame(r *bytes.Reader, _ protocol.VersionNumber) (*FecRespFrame, error) {
	_, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	streamID, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}

	offset, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}

	frame := &FecRespFrame{}
	frame.StreamID = protocol.StreamID(streamID)
	frame.StreamOffset = protocol.ByteCount(offset)

	return frame, nil
}

// Write writes a FEC frame
func (f *FecRespFrame) Write(b *bytes.Buffer, version protocol.VersionNumber) error {
	/*if len(f.FecData) == 0 && !f.Fin {
		return errors.New("FECFrame: attempting to write empty frame without FIN")
	} */

	//fmt.Printf("yunmin - fec_frame.go: Write() \n")
	typeByte := byte(0x20)
	// if f.Fin {
	// 	typeByte ^= 0x01
	// }
	b.WriteByte(typeByte)
	quicvarint.Write(b, uint64(f.StreamID))
	quicvarint.Write(b, uint64(f.StreamOffset))
	quicvarint.Write(b, uint64(f.PacketLossRate))

	return nil
}

// Length returns the total length of the FEC frame
/*func (f *FecRespFrame) Length(version protocol.VersionNumber) protocol.ByteCount {
	length := (1 + quicvarint.Len(uint64(f.StreamID)) + quicvarint.Len(uint64(f.StreamOffset)) +
		quicvarint.Len(uint64(f.FecDataOffset)) + quicvarint.Len(uint64(f.SymbolSize)) +
		quicvarint.Len(uint64(f.NumOfSymbols)) + quicvarint.Len(uint64(f.FecDataLength)) +
		quicvarint.Len(uint64(f.FecTotalLength)))

	return length + f.FecDataLen()
}*/

// FecDataLen gives the length of data in bytes
//func (f *FecRespFrame) FecDataLen() protocol.ByteCount {
//	return protocol.ByteCount(len(f.FecData))
//}

// // MaxDataLen returns the maximum data length
// // If 0 is returned, writing will fail (a FEC frame must contain at least 1 byte of data).
// func (f *FecFrame) MaxDataLen(maxSize protocol.ByteCount, version protocol.VersionNumber) protocol.ByteCount {
// 	headerLen := 1 + quicvarint.Len(uint64(f.StreamID))
// 	if f.Offset != 0 {
// 		headerLen += quicvarint.Len(uint64(f.Offset))
// 	}
// 	if f.DataLenPresent {
// 		// pretend that the data size will be 1 bytes
// 		// if it turns out that varint encoding the length will consume 2 bytes, we need to adjust the data length afterwards
// 		headerLen++
// 	}
// 	if headerLen > maxSize {
// 		return 0
// 	}
// 	maxDataLen := maxSize - headerLen
// 	if f.DataLenPresent && quicvarint.Len(uint64(maxDataLen)) != 1 {
// 		maxDataLen--
// 	}
// 	return maxDataLen
// }

// // MaybeSplitOffFrame splits a frame such that it is not bigger than n bytes.
// // It returns if the frame was actually split.
// // The frame might not be split if:
// // * the size is large enough to fit the whole frame
// // * the size is too small to fit even a 1-byte frame. In that case, the frame returned is nil.
// func (f *StreamFrame) MaybeSplitOffFrame(maxSize protocol.ByteCount, version protocol.VersionNumber) (*StreamFrame, bool /* was splitting required */) {
// 	if maxSize >= f.Length(version) {
// 		return nil, false
// 	}

// 	n := f.MaxDataLen(maxSize, version)
// 	if n == 0 {
// 		return nil, true
// 	}

// 	new := GetStreamFrame()
// 	new.StreamID = f.StreamID
// 	new.Offset = f.Offset
// 	new.Fin = false
// 	new.DataLenPresent = f.DataLenPresent

// 	// swap the data slices
// 	new.Data, f.Data = f.Data, new.Data
// 	new.fromPool, f.fromPool = f.fromPool, new.fromPool

// 	f.Data = f.Data[:protocol.ByteCount(len(new.Data))-n]
// 	copy(f.Data, new.Data[n:])
// 	new.Data = new.Data[:n]
// 	f.Offset += n

// 	return new, true
// }

// func (f *StreamFrame) PutBack() {
// 	putStreamFrame(f)
// }
