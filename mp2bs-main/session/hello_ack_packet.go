package session

import (
	"bytes"
	"mp2bs/config"
	"mp2bs/util"
)

const HELLO_ACK_PACKET_HEADER_LEN = 8

type NicInfo struct {
	Type    byte
	AddrLen byte
	Addr    []byte
}

type HelloAckPacket struct {
	Type      byte
	Length    uint16
	SessionID uint32
	NumPath   byte
	NicInfos  []NicInfo
}

func CreateHelloAckPacket(sessionID uint32, nicInfos []NicInfo) *HelloAckPacket {
	packet := HelloAckPacket{}
	packet.Type = config.Conf.HELLO_ACK_PACKET
	packet.SessionID = sessionID
	packet.NumPath = byte(len(nicInfos))
	packet.NicInfos = nicInfos
	nicInfoLen := 0
	for _, nicInfo := range nicInfos {
		nicInfoLen += int(nicInfo.AddrLen + 2)
	}
	packet.Length = uint16(HELLO_ACK_PACKET_HEADER_LEN + nicInfoLen)

	return &packet
}

func ParseHelloAckPacket(r *bytes.Reader) (*HelloAckPacket, error) {
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

	nicInfos := make([]NicInfo, numPath)
	for i := 0; i < len(nicInfos); i++ {
		nicInfos[i].Type, err = r.ReadByte()
		if err != nil {
			return nil, err
		}
		nicInfos[i].AddrLen, err = r.ReadByte()
		if err != nil {
			return nil, err
		}

		nicInfos[i].Addr = make([]byte, nicInfos[i].AddrLen)
		for j := 0; j < int(nicInfos[i].AddrLen); j++ {
			nicInfos[i].Addr[j], err = r.ReadByte()
			if err != nil {
				return nil, err
			}
		}
	}

	packet := &HelloAckPacket{}
	packet.Type = packetType
	packet.Length = packetLegnth
	packet.SessionID = sessionID
	packet.NumPath = numPath
	packet.NicInfos = nicInfos

	return packet, nil
}

// Write Hello ACK Packet
func (p *HelloAckPacket) Write(b *bytes.Buffer) error {
	b.WriteByte(p.Type)
	util.WriteUint16(b, uint16(p.Length))
	util.WriteUint32(b, uint32(p.SessionID))
	b.WriteByte(p.NumPath)

	for i := 0; i < int(p.NumPath); i++ {
		b.WriteByte(p.NicInfos[i].Type)
		b.WriteByte(p.NicInfos[i].AddrLen)
		for j := 0; j < int(p.NicInfos[i].AddrLen); j++ {
			b.WriteByte(p.NicInfos[i].Addr[j])
		}
	}

	return nil
}
