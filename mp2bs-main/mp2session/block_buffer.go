package mp2session

type BlockBuffer struct {
	lastRecvBytes uint32
	readBytes     uint32
	buffer        []*Segment
}

func CreateBlockBuffer() *BlockBuffer {
	b := BlockBuffer{
		lastRecvBytes: 0,
		readBytes:     0,
		buffer:        make([]*Segment, 0),
	}

	return &b
}

func (b *BlockBuffer) PushSegment(segment *Segment) {
	b.buffer = append(b.buffer, segment)
}

func (b *BlockBuffer) ReadSegment() *Segment {
	return b.buffer[0]
}

func (b *BlockBuffer) RemoveSegment() {
	b.buffer = b.buffer[1:]
}

func (b *BlockBuffer) IsEmpty() bool {
	return (len(b.buffer) == 0)
}

func (b *BlockBuffer) GetLength() int {
	return len(b.buffer)
}
