package mp2session

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"mp2bs/config"
	"mp2bs/session"
	"mp2bs/util"
	"sync"
	"time"
)

type Mp2Session struct {
	mutexControl      sync.Mutex
	sessionMap        map[uint32]*session.Session
	blockOwners       map[uint32]uint32
	blockNumber       uint32
	mp2SessionManager *Mp2SessionManager
	controlRecvBuffer []*ControlPacket
	sem               chan uint32
	key               uint32
	goodbye           bool
	raptor            *Raptor
	firstSegment      bool
	noTimeoutChan     chan bool
}

func CreateMp2Session(sess *session.Session, mp2SessManager *Mp2SessionManager) (*Mp2Session, error) {
	s := Mp2Session{
		sessionMap:        make(map[uint32]*session.Session),
		blockOwners:       make(map[uint32]uint32),
		mp2SessionManager: mp2SessManager,
		controlRecvBuffer: make([]*ControlPacket, 0),
		noTimeoutChan:     make(chan bool, 1),
		goodbye:           false,
		firstSegment:      true,
	}

	// Add a session if sender
	if sess != nil {
		s.AddSession(sess)
	}

	if config.FEC_MODE {
		s.raptor = mp2SessManager.raptor
	}

	return &s, nil
}

func (s *Mp2Session) AddSession(session *session.Session) {
	if session != nil {
		s.sessionMap[session.SessionID] = session
		s.blockOwners[session.SessionID] = config.PEER_NOT_KNOWN
	}
}

// Send a Block Find Packet
func (s *Mp2Session) FindBlock(sessionID uint32, blockNumber uint32) {
	util.Log("Mp2Session.FindBlock(): SessionID=%d, BlockNumber=%d", sessionID, blockNumber)

	s.blockNumber = blockNumber

	// Reset block owner list
	s.blockOwners[sessionID] = config.PEER_NOT_KNOWN

	s.sendBlockFindPacket(sessionID, blockNumber)
}

// Send a Block Info Packet
func (s *Mp2Session) BlockInfo(blockNumber uint32, blockSize uint32, status uint32) {
	util.Log("Mp2Session.BlockInfo(): BlockNumber=%d, BlockSize=%d, Status=%d", blockNumber, blockSize, status)

	s.blockNumber = blockNumber

	for sessionID := range s.sessionMap {
		s.sendBlockInfoPacket(sessionID, blockNumber, blockSize, status)
	}
}

// Send a Node Info Packet
//
// It will be called by anchor node for testing.
func (s *Mp2Session) SendNodeInfo(sessionID uint32) {
	nodeInfosLen := len(config.Conf.PEER_ADDRS)
	nodeInfos := make([]NodeInfo, nodeInfosLen)

	for i := 0; i < len(nodeInfos); i++ {
		nodeInfos[i].NodeIP = util.Ip2int(config.Conf.PEER_ADDRS[i])
		nodeInfos[i].Port = config.Conf.PEER_PORTS[i]
		nodeInfos[i].NumOfChilds = config.Conf.NUM_OF_CHILDS[i]
		nodeInfos[i].OffsetOfChild = config.Conf.OFFSET_OF_CHILDS[i]
	}

	s.sendNodeInfoPacket(sessionID, nodeInfos)
}

// Start mp2session handler
func (s *Mp2Session) HandlerStart(sessionID uint32) {
	go s.sessionHandler(sessionID)
}

// Receive packets and Handle received packet
func (s *Mp2Session) sessionHandler(sessionID uint32) {
	util.Log("Mp2Session.sessionHandler(): Session handler started (SessionID=%d)", sessionID)

	session := s.sessionMap[sessionID]

	for {
		buf := make([]byte, config.Conf.PACKET_SIZE)

		// Read packet type and length
		//_, err := session.Read(buf[:3])
		_, err := session.ReadWithSize(buf[:3], 3)
		if err != nil {
			panic(err)
		}

		r := bytes.NewReader(buf[:3])
		packetType, _ := r.ReadByte()
		packetLength, _ := util.ReadUint16(r)

		if packetType == 0 && packetLength == 0 {
			util.Log("Mp2Session.sessionHandler(): SessionID=%d, Close!!!", sessionID)
			break
		}

		/*
			if packetLength > 1500 || packetLength == 0 {
				b := make([]byte, 10)
				session.Read(b)
				util.Log("%d %d %d %d %d %d %d %d %d %d ", b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7], b[8], b[9])
				panic("Mp2Session.sessionHandler(): Wrong packet length!")
			}
		*/

		_, err = session.ReadWithSize(buf[3:packetLength], int(packetLength-3)) // Read after field of packet length

		// util.Log("len=%d / %d %d %d %d ", readLen, buf[3], buf[4], buf[5], buf[6])
		// if packetLength > 1024 {
		// 	util.Log("%d %d %d %d %d %d %d %d %d %d ", buf[packetLength-1], buf[packetLength-9], buf[packetLength-8], buf[packetLength-7], buf[packetLength-6], buf[packetLength-5], buf[packetLength-4], buf[packetLength-3], buf[packetLength-2], buf[packetLength-1])
		// }
		if err != nil {
			panic(err)
		}

		// Parse packet
		s.parsePacket(packetType, buf, session.SessionID)
	}
}

func (s *Mp2Session) parsePacket(packetType byte, packet []byte, sessionID uint32) {
	r := bytes.NewReader(packet)

	switch packetType {
	// Block Find Packet
	case config.BLOCK_FIND_PACKET:
		packet, err := ParseBlockFindPacket(r)
		if err != nil {
			panic(err)
		}
		s.handleBlockFindPacket(packet)
	// Block Info Packet
	case config.BLOCK_INFO_PACKET:
		packet, err := ParseBlockInfoPacket(r)
		if err != nil {
			panic(err)
		}
		s.handleBlockInfoPacket(packet, sessionID)
	// Block Request Packet
	case config.BLOCK_REQUEST_PACKET:
		packet, err := ParseBlockRequestPacket(r)
		if err != nil {
			panic(err)
		}
		s.handleBlockRequestPacket(packet, sessionID)
	// Block Data Packet
	case config.BLOCK_DATA_PACKET:
		packet, err := ParseBlockDataPacket(r)
		if err != nil {
			panic(err)
		}
		// Send no timeout
		s.noTimeoutChan <- true
		s.handleBlockDataPacket(packet)
	// Block Data ACK Packet
	case config.BLOCK_DATA_ACK_PACKET:
		packet, err := ParseBlockDataAckPacket(r)
		if err != nil {
			panic(err)
		}
		s.handleBlockDataAckPacket(packet)
	// Block FIN Packet
	case config.BLOCK_FIN_PACKET:
		packet, err := ParseBlockFinPacket(r)
		if err != nil {
			panic(err)
		}
		s.handleBlockFinPacket(packet, sessionID)
	// Control Packet
	case config.CONTROL_PACKET:
		packet, err := ParseControlPacket(r)
		if err != nil {
			panic(err)
		}
		s.handleControlPacket(packet)
	// Node Info Packet
	case config.NODE_INFO_PACKET:
		packet, err := ParseNodeInfoPacket(r)
		if err != nil {
			panic(err)
		}
		s.handleNodeInfoPacket(packet, sessionID)
	// Node Info ACK Packet
	case config.NODE_INFO_ACK_PACKET:
		packet, err := ParseNodeInfoAckPacket(r)
		if err != nil {
			panic(err)
		}
		s.handleNodeInfoAckPacket(packet, sessionID)
	default:
		panic(fmt.Sprintf("Mp2Session.parsePacket(): Unknown packet type (%d)", packetType))
	}
}

// Handle a Block Find Packet
func (s *Mp2Session) handleBlockFindPacket(packet *BlockFindPacket) {

	s.mp2SessionManager.blockNumChannel[s.key] <- packet.BlockNumber

	util.Log("Mp2Session.handleBlockFindPacket(): Mp2sessionID=%d, BlockNumber=%d", s.key, packet.BlockNumber)
}

// Handle a Block Info Packet
func (s *Mp2Session) handleBlockInfoPacket(packet *BlockInfoPacket, sessionID uint32) {
	util.Log("Mp2Session.handleBlockInfoPacket(): BlockNumber=%d, Status=%d, BlockSize=%d ", packet.BlockNumber, packet.Status, packet.BlockSize)

	s.blockNumber = packet.BlockNumber

	// Set status
	s.blockOwners[sessionID] = config.PEER_HAS_BLOCK

	// Check whether all senders send Block Info Packet
	cnt := 0
	for _, status := range s.blockOwners {
		if status == config.PEER_HAS_BLOCK || status == config.PEER_HAS_NO_BLOCK {
			cnt++
		}
	}

	// TODO: now we use a synchronous request scheme but we have to use an asynchronous request scheme
	// If Block Info Packets are arrived from all peers
	if cnt == len(s.blockOwners) {

		s.mp2SessionManager.blockInfoChannel[s.key] <- packet

		util.Log("Mp2Session.handleBlockInfoPacket(): Mp2session ID=%d, BlockNumber=%d, BlockOwners=%d", s.key, s.blockNumber, len(s.blockOwners))

		// We assume that block buffer is created at the first time
		if s.mp2SessionManager.recvBuffer[s.blockNumber] == nil {
			// Create a Block buffer
			s.mp2SessionManager.recvBuffer[s.blockNumber] = CreateBlockBuffer()

			util.Log("Mp2Session.handleBlockInfoPacket(): Create block buffer!!")
		}

		blockBuffer := s.mp2SessionManager.recvBuffer[s.blockNumber]

		if !config.FEC_MODE {
			go s.sendBlockRequest(packet.BlockNumber, packet.BlockSize, blockBuffer.lastRecvBytes)
		} else {
			go s.sendFecBlockRequest(packet.BlockNumber, packet.BlockSize)
		}

		go s.timer()

	}
}

// Handle a Block Request Packet
func (s *Mp2Session) handleBlockRequestPacket(packet *BlockRequestPacket, sessionID uint32) {
	util.Log("Mp2Session.handleBlockRequestPacket(): SessionID=%d, BlockNumber=%d, ReqDataNumber=%d-%d, FecIndex=%d ",
		sessionID, packet.BlockNumber, packet.StartDataNumber, packet.EndDataNumber, packet.FecIndex)

	go s.sendBlockData(sessionID, packet.BlockNumber, packet.StartDataNumber, packet.EndDataNumber, packet.FecIndex)

	if config.FEC_MODE {
		go s.sendBlockRedData(sessionID, packet.BlockNumber, packet.StartDataNumber, packet.EndDataNumber, packet.FecIndex)
	}
}

// Handle a Block Data Packet
func (s *Mp2Session) handleBlockDataPacket(packet *BlockDataPacket) {
	// Find proper segment
	// TODO: change to map structure?
	var segment *Segment
	segment = nil
	blockBuffer := s.mp2SessionManager.recvBuffer[s.blockNumber]

	util.Log("Mp2Session.handleBlockDataPacket(): BlockNumber=%d / DataNumber=%d / SegmentList=%d / DataLen=%d",
		packet.BlockNumber, packet.DataNumber, blockBuffer.GetLength(), len(packet.Data))

	for i := 0; i < blockBuffer.GetLength(); i++ {
		if packet.Redundant == config.FEC_SOURCE_DATA {
			if packet.BlockNumber == blockBuffer.buffer[i].BlockNumber &&
				packet.DataNumber >= blockBuffer.buffer[i].startDataNumber &&
				packet.DataNumber <= blockBuffer.buffer[i].endDataNumber {
				segment = blockBuffer.buffer[i]
				break
			}
		} else {
			if packet.BlockNumber == blockBuffer.buffer[i].BlockNumber &&
				packet.DataNumber >= blockBuffer.buffer[i].startRedDataNumber &&
				packet.DataNumber <= blockBuffer.buffer[i].endRedDataNumber {
				segment = blockBuffer.buffer[i]
				break
			}
		}
	}

	if segment == nil {
		util.Log("Mp2Session.handleBlockDataPacket(): Segment has been already popped")
		return
	}

	// Store received data into segment
	if packet.Redundant == config.FEC_SOURCE_DATA {
		segment.PushData(packet.BlockNumber, packet.DataNumber, packet.Data)

		// Forwarding
		s.mp2SessionManager.AddSendBuffer(packet.Data)

		blockBuffer.lastRecvBytes += uint32(len(packet.Data))
	} else {
		segment.PushRedData(packet.BlockNumber, packet.DataNumber, packet.Data)
	}

	// TODO: go routine?
	/* Check FEC decoding
	if config.FEC_MODE && !segment.Complete {
		// s.segmentList[idx].FecDecode(s.raptor)
	}
	*/

	// Push session ID to start block request for next segment
	if segment.Complete && !segment.FecFinish {
		s.sem <- segment.SessionID
		segment.FecFinish = true

		util.Log("Mp2Session.handleBlockDataPacket(): Segment Complete!!!!!!!!!!!!!!!!!!!!")
	}
}

// Handle a Block Data ACK Packet
func (s *Mp2Session) handleBlockDataAckPacket(packet *BlockDataAckPacket) {
	// TODO: Is this necessary?
	util.Log("Mp2Session.handleBlockDataAckPacket(): BlockNumber=%d, LastDataNumber=%d", packet.BlockNumber, packet.LastDataNumber)
}

// Handle a Block FIN Packet
func (s *Mp2Session) handleBlockFinPacket(packet *BlockFinPacket, sessionID uint32) {
	util.Log("Mp2Session.handleBlockFinPacket(): BlockNumber=%d", packet.BlockNumber)

	s.mp2SessionManager.blockFinishFlag[s.key] <- true
}

// Handle a Control Packet
func (s *Mp2Session) handleControlPacket(packet *ControlPacket) {
	util.Log("Mp2Session.handleControlPacket(): Length=%d, Data=%s", len(packet.Data), packet.Data)

	// Push Control Packet into control recvBuffer
	s.mutexControl.Lock()
	s.controlRecvBuffer = append(s.controlRecvBuffer, packet)
	s.mutexControl.Unlock()
}

// Handle a Node Info Packet
func (s *Mp2Session) handleNodeInfoPacket(packet *NodeInfoPacket, sessionID uint32) {
	util.Log("Mp2Session.handleNodeInfoPacket(): Number of nodes=%d", len(packet.NodeInfos))

	// Set node information to mp2session manager
	s.mp2SessionManager.Mp2SetNodeInfo(packet.NodeInfos)

	// If there are child nodes, forward Node Info Packet
	s.mp2SessionManager.Mp2SendNodeInfo(packet.NodeInfos)

	// Send Node Info ACK Packet
	s.sendNodeInfoAckPacket(sessionID, uint16(len(packet.NodeInfos)))
}

// Handle a Node Info ACK Packet
func (s *Mp2Session) handleNodeInfoAckPacket(packet *NodeInfoAckPacket, sessionID uint32) {
	util.Log("Mp2Session.handleNodeInfoAckPacket(): Number of nodes=%d", packet.NumOfInfo)

	s.mp2SessionManager.nodeInfoFlag[sessionID] = true

	for id, flag := range s.mp2SessionManager.nodeInfoFlag {
		util.Log("Mp2Session.handleNodeInfoAckPacket(): sessionID=%d, p.nodeInfoFlag=%t", id, flag)

		// All Node Info Packet are not received from child peers
		if !flag {
			return
		}
	}

	// All Node Info Packet are received from from child peers
	// Now we can send Block Info Packet
	if len(s.mp2SessionManager.nodeInfoChan) == 0 {
		s.mp2SessionManager.nodeInfoChan <- true
	}
}

// Send a Block Request Packet
func (s *Mp2Session) sendBlockRequest(blockNumber uint32, blockSize uint32, offset uint32) {
	numPeers := 0
	size := uint32(0)

	// Count the number of peers which has requested block
	for _, status := range s.blockOwners {
		if status == config.PEER_HAS_BLOCK {
			numPeers++
		}
	}

	// Create a bounded channel for a semaphore
	s.sem = make(chan uint32, numPeers)

	// Push available peers (push sessionID)
	for sessionID, status := range s.blockOwners {
		if status == config.PEER_HAS_BLOCK {
			s.sem <- sessionID
		}
	}

	// Repeat until the entire block is requested
	idx := uint32(0)
	for offset < blockSize {
		selectedSessionID := <-s.sem

		// Create a segment
		if blockSize-offset >= config.SEGMENT_SIZE {
			size = config.SEGMENT_SIZE
		} else {
			size = blockSize - offset
		}

		util.Log("Mp2Session.sendBlockRequest(): offset=%d, size=%d", offset, size)

		// Request (set block number and range)
		s.blockOwners[selectedSessionID] = config.PEER_DOWNLOADING

		startDataNumber := offset / config.PAYLOAD_SIZE
		numData := uint32(math.Ceil(float64(size) / float64(config.PAYLOAD_SIZE)))
		endDataNumber := startDataNumber + numData - 1
		lastSegment := false
		if offset+size == blockSize {
			lastSegment = true
		}

		// Create a segment for receiving
		segment, _ := CreateSegment(blockNumber, selectedSessionID, idx, startDataNumber, endDataNumber, size, lastSegment)
		s.mp2SessionManager.AddRecvBuffer(segment)

		util.Log("Mp2Session.sendBlockRequest(): SessionID=%d, NumPeer=%d, BlockNumber=%d, Offset=%d-%d, DataNumber=%d-%d",
			selectedSessionID, numPeers, blockNumber, offset, offset+size-1, startDataNumber, endDataNumber)

		s.sendBlockRequestPacket(selectedSessionID, blockNumber, startDataNumber, endDataNumber, 0)

		offset += size
		idx++
	}
}

// Send a FEC Block Request Packet (FEC Mode)
func (s *Mp2Session) sendFecBlockRequest(blockNumber uint32, blockSize uint32) {
	numPeers := 0
	offset := uint32(0)
	size := uint32(0)

	// Count the number of peers which has requested block
	for _, status := range s.blockOwners {
		if status == config.PEER_HAS_BLOCK {
			numPeers++
		}
	}

	// Create a bounded channel for a semaphore
	s.sem = make(chan uint32, 1)

	// Push available peers (push sessionID)
	for sessionID, status := range s.blockOwners {
		if status == config.PEER_HAS_BLOCK {
			s.sem <- sessionID
			break // TODO chanage
		}
	}

	// Repeat until the entire block is requested
	fecIdx := uint32(0)
	for offset < blockSize {
		// TODO: Wait until receive complete
		selectedSessionID := <-s.sem

		// Create a segment
		if blockSize-offset >= config.SEGMENT_SIZE {
			size = config.SEGMENT_SIZE
		} else {
			size = blockSize - offset
		}

		startDataNumber := offset / config.PAYLOAD_SIZE
		numData := uint32(math.Ceil(float64(size) / float64(config.PAYLOAD_SIZE)))
		endDataNumber := startDataNumber + numData - 1
		lastSegment := false
		if offset+size == blockSize {
			lastSegment = true
		}

		// Create a FEC segment for receiving
		segment, _ := CreateSegmentWithRedundant(blockNumber, selectedSessionID, fecIdx, startDataNumber, endDataNumber, size, lastSegment,
			s.raptor.SymbolSize, s.raptor.NumSrcSymbols, s.raptor.NumEncSymbols)
		s.mp2SessionManager.AddRecvBuffer(segment)

		// Send a Block Request Packet to all peers
		fecIdx = uint32(numPeers * 10)
		for sessionID, status := range s.blockOwners {
			if status == config.PEER_HAS_BLOCK {
				util.Log("Mp2Session.sendFecBlockRequest(): SessionID=%d, BlockNumber=%d, Offset=%d-%d, DataNumber=%d-%d",
					sessionID, blockNumber, offset, offset+size-1, startDataNumber, endDataNumber)
				s.sendBlockRequestPacket(sessionID, blockNumber, startDataNumber, endDataNumber, byte(fecIdx))
				fecIdx++
			}
		}

		offset += size
	}
}

func (s *Mp2Session) sendBlockData(sessionID uint32, blockNumber uint32, startDataNumber uint32, endDataNumber uint32, fecIndex byte) {

	numPeers := uint32(fecIndex / 10)
	peerIdx := uint32(fecIndex - byte(numPeers*10))

	// Wait until the data corresponding to endDataNumber is available
	lastOffset := endDataNumber * config.PAYLOAD_SIZE
	for uint32(len(s.mp2SessionManager.sendBuffer)) < lastOffset {
	}

	// TODO: hard coded no transmission
	if s.firstSegment && s.mp2SessionManager.stopSendCond > 0 {
		s.firstSegment = false
	}

	blockBuffer := s.mp2SessionManager.recvBuffer[s.blockNumber]

	if !s.firstSegment {
		if blockBuffer.lastRecvBytes >= s.mp2SessionManager.stopSendCond {
			util.Log("Mp2Session.sendBlockData(): Don't send block data!!!!!!!!!!!!! (lastRecvBytes=%d)", blockBuffer.lastRecvBytes)
			return
		}
	}

	// Send a block data
	for dataNumber := startDataNumber; dataNumber <= endDataNumber; dataNumber++ {
		if config.FEC_MODE && dataNumber%numPeers != peerIdx {
			continue
		}

		// Jump to startDataNumber
		offset := dataNumber * config.PAYLOAD_SIZE
		size := uint32(0)
		if offset+uint32(config.PAYLOAD_SIZE) <= uint32(len(s.mp2SessionManager.sendBuffer)) {
			size = uint32(config.PAYLOAD_SIZE)
		} else {
			size = (uint32(len(s.mp2SessionManager.sendBuffer)) - offset)
		}

		util.Log("Mp2Session.sendBlockData(): Offset=%d, Size=%d, Length of send buffer=%d", offset, size, len(s.mp2SessionManager.sendBuffer))

		data := make([]byte, size)
		copy(data, s.mp2SessionManager.sendBuffer[offset:offset+size])

		// Send a Block Data Packet
		s.sendBlockDataPacket(sessionID, blockNumber, dataNumber, config.FEC_SOURCE_DATA, data)
	}
}

func (s *Mp2Session) sendBlockRedData(sessionID uint32, blockNumber uint32, startDataNumber uint32, endDataNumber uint32, fecIndex byte) {

	// Make source block
	srcBlock := make([]byte, config.SEGMENT_SIZE)
	offset := startDataNumber * config.PAYLOAD_SIZE
	size := uint32(0)
	if offset+uint32(config.SEGMENT_SIZE) <= uint32(len(s.mp2SessionManager.sendBuffer)) {
		size = uint32(config.SEGMENT_SIZE)
	} else {
		size = (uint32(len(s.mp2SessionManager.sendBuffer)) - offset)
	}
	copy(srcBlock, s.mp2SessionManager.sendBuffer[offset:offset+size])

	// Raptor Encoding for segment
	redundant := s.raptor.Encode(srcBlock)
	if len(redundant) == 0 {
		panic(fmt.Sprintf("Mp2Session.sendBlockRedData(): Raptor Encode Fail! (Block=%d-%d)", offset, offset+size))
	}

	// Send redundant data
	numPeers := uint32(fecIndex / 10)
	peerIdx := uint32(fecIndex - byte(numPeers*10))
	startRedDataNumber := endDataNumber + 1
	endRedDataNumber := endDataNumber + uint32(math.Ceil(float64(len(redundant))/float64(config.PAYLOAD_SIZE)))
	for dataNumber := startRedDataNumber; dataNumber <= endRedDataNumber; dataNumber++ {
		if config.FEC_MODE && dataNumber%numPeers != peerIdx {
			continue
		}

		redOffset := (dataNumber - startRedDataNumber) * config.PAYLOAD_SIZE
		redSize := uint32(config.PAYLOAD_SIZE)
		util.Log("Mp2Session.sendBlockRedData(): Offset=%d, Size=%d, Length of redundant=%d", redOffset, redSize, len(redundant))

		data := make([]byte, redSize)
		copy(data, redundant[redOffset:redOffset+redSize])

		// Send a Block Data Packet
		s.sendBlockDataPacket(sessionID, blockNumber, dataNumber, config.FEC_REDUNDANT_DATA, data)
	}
}

// Send a Block Data ACK Packet
func (s *Mp2Session) sendBlockDataAckPacket(sessionID uint32, blockNumber uint32, lastDataNumber uint32) {

	packet := CreateBlockDataAckPacket(blockNumber, lastDataNumber)
	util.Log("Mp2Session.sendBlockDataAckPacket(): SessionID=%d, BlockNumber=%d, lastDataNumber=%d", sessionID, blockNumber, lastDataNumber)

	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(sessionID, b.Bytes(), packet.Type)
}

// Send a Block Find Packet
func (s *Mp2Session) sendBlockFindPacket(sessionID uint32, blockNumber uint32) {

	packet := CreateBlockFindPacket(blockNumber)
	util.Log("Mp2Session.sendBlockFindPacket(): SessionID=%d, BlockNumber=%d", sessionID, blockNumber)

	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(sessionID, b.Bytes(), packet.Type)
}

// Send a Block Info Packet
func (s *Mp2Session) sendBlockInfoPacket(sessionID uint32, blockNumber uint32, blockSize uint32, status uint32) {

	// TODO: we assume that sender peer has a requested block always
	packet := CreateBlockInfoPacket(blockNumber, status, blockSize)
	util.Log("Mp2Session.sendBlockInfoPacket(): SessionID=%d, BlockNumber=%d, BlockSize=%d, Status=%d", sessionID, blockNumber, blockSize, status)

	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(sessionID, b.Bytes(), packet.Type)
}

// Send a Block Request Packet
func (s *Mp2Session) sendBlockRequestPacket(sessionID uint32, blockNumber uint32, startDataNumber uint32, endDataNumber uint32, fecIndex byte) {

	packet := CreateBlockRequestPacket(blockNumber, startDataNumber, endDataNumber, fecIndex)
	util.Log("Mp2Session.sendBlockRequestPacket(): SessionID=%d, BlockNumber=%d, DataNumber=%d-%d", sessionID, blockNumber, startDataNumber, endDataNumber)

	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(sessionID, b.Bytes(), packet.Type)
}

// Send a Block Data Packet
func (s *Mp2Session) sendBlockDataPacket(sessionID uint32, blockNumber uint32, dataNumber uint32, redundant byte, data []byte) {

	packet := CreateBlockDataPacket(blockNumber, dataNumber, sessionID, redundant, data)
	util.Log("Mp2Session.sendBlockDataPacket(): SessionID=%d, BlockNumber=%d, DataNumber=%d, Length of data=%d", sessionID, blockNumber, dataNumber, len(data))

	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(sessionID, b.Bytes(), packet.Type)
}

// Send a Block FIN Packet
func (s *Mp2Session) sendBlockFinPacket(sessionID uint32) {

	packet := CreateBlockFinPacket(sessionID, s.blockNumber)
	util.Log("Mp2Session.sendBlockFinPacket(): SessionID=%d, BlockNumber=%d", sessionID, s.blockNumber)

	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(sessionID, b.Bytes(), packet.Type)
}

// Send a Control Packet
func (s *Mp2Session) sendControlPacket(sessionID uint32, data []byte) {

	packet := CreateControlPacket(data)
	util.Log("Mp2Session.sendControlPacket(): SessionID=%d, Length of data=%d", sessionID, len(data))

	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(sessionID, b.Bytes(), packet.Type)
}

// Send a Node Info Packet
func (s *Mp2Session) sendNodeInfoPacket(sessionID uint32, nodeInfos []NodeInfo) {

	packet := CreateNodeInfoPacket(nodeInfos)
	util.Log("Mp2Session.sendNodeInfoPacket(): SessionID=%d, Number of nodes=%d", sessionID, len(nodeInfos))

	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(sessionID, b.Bytes(), packet.Type)
}

// Send a Node Info ACK Packet
func (s *Mp2Session) sendNodeInfoAckPacket(sessionID uint32, numOfInfo uint16) {

	packet := CreateNodeInfoAckPacket(numOfInfo)
	util.Log("Mp2Session.sendNodeInfoAckPacket(): SessionID=%d, Number of nodes=%d", sessionID, numOfInfo)

	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(sessionID, b.Bytes(), packet.Type)
}

// Send a packet
func (s *Mp2Session) sendPacket(sessionID uint32, buf []byte, pType byte) {
	util.Log("Mp2Session.sendPacket(): Send data len=%d (buffer size=%d)", len(buf), len(s.mp2SessionManager.sendBuffer))

	session := s.sessionMap[sessionID]

	_, err := session.Write(buf, pType)
	if err != nil {
		log.Println(err)
	}
}

// Read a Control Packet
func (s *Mp2Session) ReadControl(buf []byte) int {
	// TODO: infinite-loop
	for len(s.controlRecvBuffer) == 0 && !s.goodbye {
	}

	readLen := 0

	if len(s.controlRecvBuffer) > 0 {
		s.mutexControl.Lock()
		// Copy packet data to buf
		packet := s.controlRecvBuffer[0]
		copy(buf, packet.Data)
		readLen = len(packet.Data)

		// Remove first packet in control receive buffer
		s.controlRecvBuffer = s.controlRecvBuffer[1:]
		s.mutexControl.Unlock()
	}

	return readLen
}

// Write a Control Packet
func (s *Mp2Session) WriteControl(buf []byte) {

	offset := uint32(0)
	buf_len := uint32(len(buf))
	sessionID := uint32(0)

	// TODO: handling session ID for Write()
	for offset < buf_len {
		size := uint32(0)
		if offset+uint32(config.PAYLOAD_SIZE) <= buf_len {
			size = uint32(config.PAYLOAD_SIZE)
		} else {
			size = (uint32(len(buf)) - offset)
		}

		data := make([]byte, size)
		copy(data, buf[offset:offset+size])

		// Send a Control Packet
		for sessionID = range s.sessionMap {
			s.sendControlPacket(sessionID, data)
		}

		offset += size
	}
}

// TODO: not only stream close, but also session close
func (s *Mp2Session) Close() {
	for _, session := range s.sessionMap {
		session.Close()
	}
}

func (s *Mp2Session) timer() {
	firstTimeout := true
	s.mp2SessionManager.mutexForTimeout.Lock()
	s.mp2SessionManager.timeoutFlag[s.blockNumber] = false
	s.mp2SessionManager.mutexForTimeout.Unlock()

timer:
	for {
		select {
		case <-s.noTimeoutChan:
			s.mp2SessionManager.mutexForTimeout.Lock()
			s.mp2SessionManager.timeoutFlag[s.blockNumber] = false
			s.mp2SessionManager.mutexForTimeout.Unlock()
		case <-time.After(config.TIMEOUT * time.Second):
			blockBuffer := s.mp2SessionManager.recvBuffer[s.blockNumber]

			util.Log("Mp2Session.timer(): Timeout!! ReadBytes=%d, LastRecvBytes=%d",
				blockBuffer.readBytes, blockBuffer.lastRecvBytes)

			if blockBuffer.lastRecvBytes == s.mp2SessionManager.blockList[s.blockNumber] {

				s.mp2SessionManager.blockFinishFlag[s.key] <- true

				for sessionID := range s.sessionMap {
					// Send a Block FIN Packet to sender side
					s.sendBlockFinPacket(sessionID)
				}

				break timer
			}

			// FIXME: not clear
			if firstTimeout {
				firstTimeout = false

				// Wait until all received bytes are read
				for blockBuffer.readBytes < blockBuffer.lastRecvBytes {
				}

				// Flush segment list
				s.mp2SessionManager.recvBuffer[s.blockNumber].buffer = nil

				// Timeout flag
				s.mp2SessionManager.mutexForTimeout.Lock()
				s.mp2SessionManager.timeoutFlag[s.blockNumber] = true
				s.mp2SessionManager.mutexForTimeout.Unlock()

				// Connect for PULL mode
				s.mp2SessionManager.Mp2ConnectForPull(s.blockNumber)
			}

		}
	}
}
