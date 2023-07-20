package mp2session

import (
	"bytes"
	config "mp2bs/config"
	util "mp2bs/util"
)

const NODE_INFO_ACK_PACKET_HEADER_LEN = 5

type NodeInfoAckPacket struct {
	Type      byte
	Length    uint16
	NumOfInfo uint16
}

func CreateNodeInfoAckPacket(NumOfInfo uint16) *NodeInfoAckPacket {
	packet := NodeInfoAckPacket{}
	packet.Type = config.NODE_INFO_ACK_PACKET
	packet.Length = NODE_INFO_ACK_PACKET_HEADER_LEN
	packet.NumOfInfo = NumOfInfo

	return &packet
}

func ParseNodeInfoAckPacket(r *bytes.Reader) (*NodeInfoAckPacket, error) {
	packetType, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	packetLegnth, err := util.ReadUint16(r)
	if err != nil {
		return nil, err
	}

	packetNumOfInfo, err := util.ReadUint16(r)
	if err != nil {
		return nil, err
	}

	packet := &NodeInfoAckPacket{}
	packet.Type = packetType
	packet.Length = packetLegnth
	packet.NumOfInfo = packetNumOfInfo

	return packet, nil
}

// Write Node Info ACK Packet
func (p *NodeInfoAckPacket) Write(b *bytes.Buffer) error {
	b.WriteByte(p.Type)
	util.WriteUint16(b, uint16(p.Length))
	util.WriteUint16(b, uint16(p.NumOfInfo))

	return nil
}
