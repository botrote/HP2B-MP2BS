package config

// TODO: define config type, read from configuration file

var Conf Config

type Config struct {
	CONFIG_USER_WRR_WEIGHT                 []uint32
	VERBOSE_MODE                           bool
	PACKET_SIZE                            uint16
	HELLO_PACKET                           byte
	HELLO_ACK_PACKET                       byte
	DATA_PACKET                            byte
	GOODBYE_PACKET                         byte
	NIC_NAMES                              []string
	NIC_ADDRS                              []string
	PEER_ADDRS                             []string
	PEER_PORTS                             []uint16
	NUM_OF_CHILDS                          []byte
	OFFSET_OF_CHILDS                       []byte
	THROUGHPUT_PERIOD                      float32
	THROUGHPUT_WEIGHT                      float32
	MULTIPATH_THRESHOLD_THROUGHPUT         float64
	MULTIPATH_THRESHOLD_SEND_COMPLETE_TIME float64
	CLOSE_TIMEOUT_PERIOD                   uint16
}

var FEC_MODE bool

const (
	BLOCK_FIND_PACKET     = 11
	BLOCK_INFO_PACKET     = 12
	BLOCK_REQUEST_PACKET  = 13
	BLOCK_DATA_PACKET     = 14
	BLOCK_DATA_ACK_PACKET = 15
	BLOCK_FIN_PACKET      = 16
	CONTROL_PACKET        = 17
	NODE_INFO_PACKET      = 18
	NODE_INFO_ACK_PACKET  = 19

	PEER_NOT_KNOWN    = 0
	PEER_HAS_BLOCK    = 1
	PEER_HAS_NO_BLOCK = 2
	PEER_DOWNLOADING  = 3

	SEGMENT_SIZE = 2 * 1024 * 1024 // 2MB
	PAYLOAD_SIZE = 1024            // 1KB

	//FEC_MODE           = false
	FEC_SOURCE_DATA    = 0
	FEC_REDUNDANT_DATA = 1

	FEC_SYMBOL_SIZE        = uint32(1024)
	FEC_NUM_SOURCE_SYMBOLS = uint32(16)
	FEC_MINIMUM_OVERHEAD   = float32(0.0758)
	FEC_CODE_RATE          = 0.8

	TIMEOUT = 2

// var SymbolSize = []uint32{1024}
// var NumberOfSrcSymbolSize = []uint32{16} //128, 192, 256, 320, 384, 448, 512 }
// var MinimumOverhead = []float32{0.0758, 0.0518, 0.0358, 0.0310, 0.0260, 0.0224, 0.0187}
// var CodeRate = 0.8
)
