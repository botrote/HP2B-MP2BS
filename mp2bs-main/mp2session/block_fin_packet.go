package mp2session

import (
	"bytes"
	config "mp2bs/config"
	util "mp2bs/util"
)

const BLOCK_FIN_PACKET_HEADER_LEN = 11

type BlockFinPacket struct {
	Type        byte
	Length      uint16
	SessionID   uint32
	BlockNumber uint32
}

func CreateBlockFinPacket(sessionID uint32, blockNumber uint32) *BlockFinPacket {
	packet := BlockFinPacket{}
	packet.Type = config.BLOCK_FIN_PACKET
	packet.Length = BLOCK_FIN_PACKET_HEADER_LEN
	packet.SessionID = sessionID
	packet.BlockNumber = blockNumber

	return &packet
}

func ParseBlockFinPacket(r *bytes.Reader) (*BlockFinPacket, error) {

	packetType, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	packetLegnth, err := util.ReadUint16(r)
	if err != nil {
		return nil, err
	}

	sessionID, err := util.ReadUint32(r)
	if err != nil {
		return nil, err
	}

	blockNumber, err := util.ReadUint32(r)
	if err != nil {
		return nil, err
	}

	packet := &BlockFinPacket{}
	packet.Type = packetType
	packet.Length = packetLegnth
	packet.SessionID = sessionID
	packet.BlockNumber = blockNumber

	return packet, nil
}

// Write Block FIN Packet
func (p *BlockFinPacket) Write(b *bytes.Buffer) error {
	b.WriteByte(p.Type)
	util.WriteUint16(b, uint16(p.Length))
	util.WriteUint32(b, uint32(p.SessionID))
	util.WriteUint32(b, uint32(p.BlockNumber))

	return nil
}
