package fec

import (
	"fmt"
	"time"

	"github.com/lucas-clemente/quic-go/internal/protocol"
)

func FecLog(format string, args ...interface{}) {
	pre := "[" + time.Now().Format(time.StampMicro) + "] "
	fmt.Printf(pre+format+"\n", args...)
}

func FecBytesLog(block []byte, length int) {
	for i := 0; i < length; i++ {
		if i%32 == 0 {
			fmt.Printf("\n%8d: ", i)
		}
		fmt.Printf("%3d ", block[i])
	}
	fmt.Printf("\n")
}

var FecRecoveredPacketList []protocol.PacketNumber
