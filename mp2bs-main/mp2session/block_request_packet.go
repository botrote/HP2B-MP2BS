package mp2session

import (
	"bytes"
	config "mp2bs/config"
	util "mp2bs/util"
)

const BLOCK_REQUEST_PACKET_HEADER_LEN = 16

type BlockRequestPacket struct {
	Type            byte
	Length          uint16
	BlockNumber     uint32
	StartDataNumber uint32
	EndDataNumber   uint32 // StartDataNumber ~ EndDataNumber (include EndDataNumber)
	FecIndex        byte   // For FEC Mode
}

func CreateBlockRequestPacket(blockNumber uint32, startDataNumber uint32, endDataNumber uint32, fecIndex byte) *BlockRequestPacket {
	packet := BlockRequestPacket{}
	packet.Type = config.BLOCK_REQUEST_PACKET
	packet.Length = BLOCK_REQUEST_PACKET_HEADER_LEN
	packet.BlockNumber = blockNumber
	packet.StartDataNumber = startDataNumber
	packet.EndDataNumber = endDataNumber
	packet.FecIndex = fecIndex
	return &packet
}

func ParseBlockRequestPacket(r *bytes.Reader) (*BlockRequestPacket, error) {

	packetType, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	packetLegnth, err := util.ReadUint16(r)
	if err != nil {
		return nil, err
	}

	blockNumber, err := util.ReadUint32(r)
	if err != nil {
		return nil, err
	}

	startDataNumber, err := util.ReadUint32(r)
	if err != nil {
		return nil, err
	}

	endDataNumber, err := util.ReadUint32(r)
	if err != nil {
		return nil, err
	}

	fecIndex, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	packet := &BlockRequestPacket{}
	packet.Type = packetType
	packet.Length = packetLegnth
	packet.BlockNumber = blockNumber
	packet.StartDataNumber = startDataNumber
	packet.EndDataNumber = endDataNumber
	packet.FecIndex = fecIndex

	return packet, nil
}

// Write Block Request Packet
func (p *BlockRequestPacket) Write(b *bytes.Buffer) error {
	b.WriteByte(p.Type)
	util.WriteUint16(b, uint16(p.Length))
	util.WriteUint32(b, uint32(p.BlockNumber))
	util.WriteUint32(b, uint32(p.StartDataNumber))
	util.WriteUint32(b, uint32(p.EndDataNumber))
	b.WriteByte(p.FecIndex)
	return nil
}
