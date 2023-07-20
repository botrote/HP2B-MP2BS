package mp2session

import (
	"bytes"
	config "mp2bs/config"
	util "mp2bs/util"
)

const BLOCK_DATA_ACK_PACKET_HEADER_LEN = 11

type BlockDataAckPacket struct {
	Type           byte
	Length         uint16
	BlockNumber    uint32
	LastDataNumber uint32
}

func CreateBlockDataAckPacket(blockNumber uint32, lastDataNumber uint32) *BlockDataAckPacket {
	packet := BlockDataAckPacket{}
	packet.Type = config.BLOCK_DATA_ACK_PACKET
	packet.Length = BLOCK_DATA_ACK_PACKET_HEADER_LEN
	packet.BlockNumber = blockNumber
	packet.LastDataNumber = lastDataNumber

	return &packet
}

func ParseBlockDataAckPacket(r *bytes.Reader) (*BlockDataAckPacket, error) {

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

	lastDataNumber, err := util.ReadUint32(r)
	if err != nil {
		return nil, err
	}

	packet := &BlockDataAckPacket{}
	packet.Type = packetType
	packet.Length = packetLegnth
	packet.BlockNumber = blockNumber
	packet.LastDataNumber = lastDataNumber

	return packet, nil
}

// Write Block Data ACK Packet
func (p *BlockDataAckPacket) Write(b *bytes.Buffer) error {
	b.WriteByte(p.Type)
	util.WriteUint16(b, uint16(p.Length))
	util.WriteUint32(b, uint32(p.BlockNumber))
	util.WriteUint32(b, uint32(p.LastDataNumber))

	return nil
}
