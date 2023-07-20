package wire

import (
	"bytes"
	"errors"
	"io"

	"github.com/lucas-clemente/quic-go/internal/protocol"
	"github.com/lucas-clemente/quic-go/quicvarint"
)

// A FEC Frame of QUIC
type FecFrame struct {
	StreamID       protocol.StreamID
	StreamOffset   protocol.ByteCount // Start of stream offset
	SymbolSize     uint32
	NumOfSymbols   uint32
	FecDataOffset  protocol.ByteCount // Start of FEC data offset
	FecDataLength  uint32
	FecTotalLength uint32
	FecData        []byte
	Fin            bool
}

func parseFecFrame(r *bytes.Reader, _ protocol.VersionNumber) (*FecFrame, error) {
	typeByte, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	fin := typeByte&0x1 > 0

	streamID, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}

	offset, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}

	temp, err := quicvarint.Read(r)
	symbolSize := uint32(temp)
	if err != nil {
		return nil, err
	}

	temp, err = quicvarint.Read(r)
	numOfSymbols := uint32(temp)
	if err != nil {
		return nil, err
	}

	fecDataOffset, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}

	temp, err = quicvarint.Read(r)
	fecDataLength := uint32(temp)
	if err != nil {
		return nil, err
	}

	temp, err = quicvarint.Read(r)
	fecTotalLength := uint32(temp)
	if err != nil {
		return nil, err
	}

	frame := &FecFrame{FecData: make([]byte, fecDataLength)}
	frame.StreamID = protocol.StreamID(streamID)
	frame.StreamOffset = protocol.ByteCount(offset)
	frame.SymbolSize = symbolSize
	frame.NumOfSymbols = numOfSymbols
	frame.FecDataOffset = protocol.ByteCount(fecDataOffset)
	frame.FecDataLength = fecDataLength
	frame.FecTotalLength = fecTotalLength
	frame.Fin = fin

	if fecDataLength != 0 {
		if _, err := io.ReadFull(r, frame.FecData); err != nil {
			return nil, err
		}
	}
	if frame.FecDataOffset+frame.FecDataLen() > protocol.MaxByteCount {
		return nil, errors.New("fec reundant data overflows maximum offset")
	}
	return frame, nil
}

// Write writes a FEC frame
func (f *FecFrame) Write(b *bytes.Buffer, version protocol.VersionNumber) error {
	if len(f.FecData) == 0 && !f.Fin {
		return errors.New("FECFrame: attempting to write empty frame without FIN")
	}

	//fmt.Printf("yunmin - fec_frame.go: Write() \n")
	typeByte := byte(0x20)
	// if f.Fin {
	// 	typeByte ^= 0x01
	// }
	b.WriteByte(typeByte)
	quicvarint.Write(b, uint64(f.StreamID))
	quicvarint.Write(b, uint64(f.StreamOffset))
	quicvarint.Write(b, uint64(f.SymbolSize))
	quicvarint.Write(b, uint64(f.NumOfSymbols))
	quicvarint.Write(b, uint64(f.FecDataOffset))
	quicvarint.Write(b, uint64(f.FecDataLength))
	quicvarint.Write(b, uint64(f.FecTotalLength))
	b.Write(f.FecData)
	return nil
}

// Length returns the total length of the FEC frame
func (f *FecFrame) Length(version protocol.VersionNumber) protocol.ByteCount {
	length := (1 + quicvarint.Len(uint64(f.StreamID)) + quicvarint.Len(uint64(f.StreamOffset)) +
		quicvarint.Len(uint64(f.FecDataOffset)) + quicvarint.Len(uint64(f.SymbolSize)) +
		quicvarint.Len(uint64(f.NumOfSymbols)) + quicvarint.Len(uint64(f.FecDataLength)) +
		quicvarint.Len(uint64(f.FecTotalLength)))

	return length + f.FecDataLen()
}

// FecDataLen gives the length of data in bytes
func (f *FecFrame) FecDataLen() protocol.ByteCount {
	return protocol.ByteCount(len(f.FecData))
}

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
