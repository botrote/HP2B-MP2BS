package session

import (
	"bytes"
	"mp2bs/config"
	"mp2bs/util"
)

const GOODBYE_PACKET_HEADER_LEN = 7

type GoodbyePacket struct {
	Type      byte
	Length    uint16
	SessionID uint32
}

func CreateGoodbyePacket(sessionID uint32) *GoodbyePacket {
	packet := GoodbyePacket{}
	packet.Type = config.Conf.GOODBYE_PACKET
	packet.Length = GOODBYE_PACKET_HEADER_LEN
	packet.SessionID = sessionID
	return &packet
}

func ParseGoodbyePacket(r *bytes.Reader) (*GoodbyePacket, error) {

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

	packet := &GoodbyePacket{}
	packet.Type = packetType
	packet.Length = packetLegnth
	packet.SessionID = sessionID

	return packet, nil
}

// Write Goodbye Packet
func (p *GoodbyePacket) Write(b *bytes.Buffer) error {
	b.WriteByte(p.Type)
	util.WriteUint16(b, uint16(p.Length))
	util.WriteUint32(b, uint32(p.SessionID))
	return nil
}
