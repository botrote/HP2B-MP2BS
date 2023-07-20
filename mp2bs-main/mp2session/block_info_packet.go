package mp2session

import (
	"bytes"
	config "mp2bs/config"
	util "mp2bs/util"
)

const BLOCK_INFO_PACKET_HEADER_LEN = 15

type BlockInfoPacket struct {
	Type        byte
	Length      uint16
	BlockNumber uint32
	Status      uint32
	BlockSize   uint32
}

func CreateBlockInfoPacket(blockNumber uint32, status uint32, blockSize uint32) *BlockInfoPacket {
	packet := BlockInfoPacket{}
	packet.Type = config.BLOCK_INFO_PACKET
	packet.Length = BLOCK_INFO_PACKET_HEADER_LEN
	packet.BlockNumber = blockNumber
	packet.Status = status
	packet.BlockSize = blockSize
	return &packet
}

func ParseBlockInfoPacket(r *bytes.Reader) (*BlockInfoPacket, error) {

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

	status, err := util.ReadUint32(r)
	if err != nil {
		return nil, err
	}

	blockSize, err := util.ReadUint32(r)
	if err != nil {
		return nil, err
	}

	packet := &BlockInfoPacket{}
	packet.Type = packetType
	packet.Length = packetLegnth
	packet.BlockNumber = blockNumber
	packet.Status = status
	packet.BlockSize = blockSize

	return packet, nil
}

// Write Block Info Packet
func (p *BlockInfoPacket) Write(b *bytes.Buffer) error {
	b.WriteByte(p.Type)
	util.WriteUint16(b, uint16(p.Length))
	util.WriteUint32(b, uint32(p.BlockNumber))
	util.WriteUint32(b, uint32(p.Status))
	util.WriteUint32(b, uint32(p.BlockSize))
	return nil
}
