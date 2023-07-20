package mp2session

import (
	"bytes"
	"fmt"
	config "mp2bs/config"
	util "mp2bs/util"
)

type ControlPacket struct {
	Type   byte
	Length uint16
	Data   []byte
}

const CONTROL_PACKET_HEADER_LEN = 3

func CreateControlPacket(data []byte) *ControlPacket {
	packet := ControlPacket{}
	packet.Type = config.CONTROL_PACKET
	packet.Length = uint16(CONTROL_PACKET_HEADER_LEN + len(data))

	if packet.Length > CONTROL_PACKET_HEADER_LEN+config.PAYLOAD_SIZE {
		panic(fmt.Sprintf("packet length is larger than maximum size! (%d>%d)", packet.Length, CONTROL_PACKET_HEADER_LEN+config.PAYLOAD_SIZE))
	}

	packet.Data = make([]byte, len(data))
	copy(packet.Data, data)
	return &packet
}

func ParseControlPacket(r *bytes.Reader) (*ControlPacket, error) {

	packetType, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	packetLegnth, err := util.ReadUint16(r)
	if err != nil {
		return nil, err
	}

	packet := &ControlPacket{}
	packet.Type = packetType
	packet.Length = packetLegnth
	packet.Data = make([]byte, packetLegnth-CONTROL_PACKET_HEADER_LEN)
	r.Read(packet.Data)

	return packet, nil
}

// Write Control Packet
func (p *ControlPacket) Write(b *bytes.Buffer) error {
	b.WriteByte(p.Type)
	util.WriteUint16(b, uint16(p.Length))
	b.Write(p.Data)

	return nil
}
