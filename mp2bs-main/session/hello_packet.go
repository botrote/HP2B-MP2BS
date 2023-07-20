package session

import (
	"bytes"
	"mp2bs/config"
	"mp2bs/util"
)

const HELLO_PACKET_HEADER_LEN = 8

type HelloPacket struct {
	Type      byte
	Length    uint16
	SessionID uint32
	NumPath   byte
}

func CreateHelloPacket(sessionID uint32, numPath byte) *HelloPacket {
	packet := HelloPacket{}
	packet.Type = config.Conf.HELLO_PACKET
	packet.Length = HELLO_PACKET_HEADER_LEN
	packet.SessionID = sessionID
	packet.NumPath = numPath

	return &packet
}

func ParseHelloPacket(r *bytes.Reader) (*HelloPacket, error) {

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

	numPath, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	packet := &HelloPacket{}
	packet.Type = packetType
	packet.Length = packetLegnth
	packet.SessionID = sessionID
	packet.NumPath = numPath

	return packet, nil
}

// Write Hello Packet
func (p *HelloPacket) Write(b *bytes.Buffer) error {
	b.WriteByte(p.Type)
	util.WriteUint16(b, uint16(p.Length))
	util.WriteUint32(b, uint32(p.SessionID))
	b.WriteByte(p.NumPath)

	return nil
}
