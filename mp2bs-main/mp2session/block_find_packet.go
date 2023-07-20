package mp2session

import (
	"bytes"
	config "mp2bs/config"
	util "mp2bs/util"
)

const BLOCK_FIND_PACKET_HEADER_LEN = 7

type BlockFindPacket struct {
	Type        byte
	Length      uint16
	BlockNumber uint32
}

func CreateBlockFindPacket(blockNumber uint32) *BlockFindPacket {
	packet := BlockFindPacket{}
	packet.Type = config.BLOCK_FIND_PACKET
	packet.Length = BLOCK_FIND_PACKET_HEADER_LEN
	packet.BlockNumber = blockNumber
	return &packet
}

func ParseBlockFindPacket(r *bytes.Reader) (*BlockFindPacket, error) {

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

	packet := &BlockFindPacket{}
	packet.Type = packetType
	packet.Length = packetLegnth
	packet.BlockNumber = blockNumber

	return packet, nil
}

// Write Block Find Packet
func (p *BlockFindPacket) Write(b *bytes.Buffer) error {
	b.WriteByte(p.Type)
	util.WriteUint16(b, uint16(p.Length))
	util.WriteUint32(b, uint32(p.BlockNumber))
	return nil
}
