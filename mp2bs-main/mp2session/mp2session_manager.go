package mp2session

import (
	"fmt"
	"io"
	"math/rand"
	"mp2bs/config"
	"mp2bs/session"
	"mp2bs/util"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

// -t means that it is a variable for testing.
// It will be removed.
type Mp2SessionManager struct {
	peerId               uint32                             // Peer ID -t
	peerPushMode         bool                               // PUSH mode flag -t
	peerModeStatus       string                             // Mode status (PUSH or PULL) -t
	myAddr               []string                           // My available addresses
	childAddr            []string                           // Child addresses
	allAddr              []string                           // All addresses
	blockList            map[uint32]uint32                  // Block list
	receivedBlockChannel chan uint32                        // Block information(block number && block size) channel
	mp2IdFromParent      uint32                             // Connection from parent
	mp2IdToChildren      uint32                             // Connection to children
	mp2IdForPull         uint32                             // Connection for pull
	mp2sessionMap        map[uint32]*Mp2Session             // Mp2session list
	sessionManager       *session.SessionManager            // Peer session manager
	raptor               *Raptor                            // Raptor
	blockNumChannel      map[uint32](chan uint32)           // Channel to check if the requested block number exists
	blockInfoChannel     map[uint32](chan *BlockInfoPacket) // Channel to check if the received block infomrmation exists
	sendBuffer           []byte                             // Send buffer
	recvBuffer           map[uint32](*BlockBuffer)          // Receive buffer (key: blockNumber, value: BlockBuffer)
	mutexForBuffer       sync.Mutex                         // Mutext for recv buffer
	mutexForTimeout      sync.Mutex                         // Mutext for timeout
	goodbye              bool                               // For reading
	timeoutFlag          map[uint32]bool                    // Timeout flag (key: blockNumber)
	nodeInfoChan         chan bool                          // Channel to check if it is okay to send a block
	nodeInfoFlag         map[uint32]bool                    // To check that all child nodes send Node Info Ack Packet (key: sessionID)
	stopSendCond         uint32                             // For pull mode test -t
	blockFinishFlag      map[uint32](chan bool)             // To check block transfer is complete
}

// CreateMp2SessionManager() function creates Mp2Session Manager instance and returns it.
func CreateMp2SessionManager(configFile string, peerID uint32) *Mp2SessionManager {

	// Parse the config file
	if _, err := toml.DecodeFile(configFile, &config.Conf); err != nil {
		panic(err)
	}

	// Create a Mp2Session Manager instance
	p := Mp2SessionManager{
		peerId:               peerID,
		myAddr:               config.Conf.NIC_ADDRS,
		blockList:            make(map[uint32]uint32),
		receivedBlockChannel: make(chan uint32, 1), // TODO: what is proper value? We don't know how many blocks we will receive.
		mp2sessionMap:        make(map[uint32]*Mp2Session),
		blockNumChannel:      make(map[uint32](chan uint32)),
		blockInfoChannel:     make(map[uint32](chan *BlockInfoPacket)),
		sendBuffer:           make([]byte, 0),
		recvBuffer:           make(map[uint32](*BlockBuffer)),
		raptor:               nil,
		goodbye:              false,
		peerPushMode:         true,
		peerModeStatus:       "",
		timeoutFlag:          make(map[uint32]bool),
		nodeInfoChan:         make(chan bool, 1),
		nodeInfoFlag:         make(map[uint32]bool),
		blockFinishFlag:      make(map[uint32](chan bool)),
	}

	// Create a Peer Session Manager
	p.sessionManager = session.CreateSessionManager(p.myAddr)

	// Create a Raptor
	if config.FEC_MODE {
		p.raptor = CreateRaptor()
	}

	return &p
}

// Mp2Connect() function supports connection with top node.
//
// It will be called by anchor node for block propagation test.
func (p *Mp2SessionManager) Mp2Connect(TopAddr string) {

	p.childAddr = append(p.childAddr, TopAddr)

	p.mp2IdToChildren = p.Mp2ConnectToChildren()
}

// Mp2ConnectToChildren() function supports connection with child nodes.
func (p *Mp2SessionManager) Mp2ConnectToChildren() uint32 {
	var err error
	var mp2sess *Mp2Session

	mp2ID := rand.Uint32()

	for _, addr := range p.childAddr {
		// Accept a peer session
		session := p.sessionManager.Connect(addr)
		util.Log("Mp2SessionManager.Mp2ConnectToChildren(): Peer Addr=%s, SessionID=%d", addr, session.SessionID)

		if p.mp2sessionMap[mp2ID] == nil {
			// Create a mp2session using the given peer session
			mp2sess, err = CreateMp2Session(session, p)
			if err != nil {
				panic(err)
			}

			util.Log("Mp2SessionManager.Mp2ConnectToChildren(): Create Mp2Session! (Mp2SessionID=%d)", mp2ID)

			mp2sess.key = mp2ID

			p.blockNumChannel[mp2ID] = make(chan uint32, 1)
			p.blockInfoChannel[mp2ID] = make(chan *BlockInfoPacket, 1)
			p.blockFinishFlag[mp2ID] = make(chan bool, 1)

			p.mp2sessionMap[mp2ID] = mp2sess
		} else {
			p.mp2sessionMap[mp2ID].AddSession(session)
		}

		p.mp2sessionMap[mp2ID].HandlerStart(session.SessionID)

		// TODO: Hard coded transmission stop -> should be removed
		if p.peerId == 1 {
			// Stop sending block data after x bytes received
			p.stopSendCond = uint32(rand.Intn(10)) * 1024 * 1024
		}
	}

	return mp2ID
}

// Mp2ConnectForPull() functions disconnects from the parent node and supports connection with other nodes for pull mode.
func (p *Mp2SessionManager) Mp2ConnectForPull(blockNumber uint32) {
	var err error
	var mp2sess *Mp2Session

	mp2ID := rand.Uint32()

	// TODO: Select five nodes?
	for idx, addr := range p.allAddr {
		// TODO: hard coding (to test PULL mode) -> should be removed
		if idx == 1 {
			continue
		}
		// Accept a peer session
		session := p.sessionManager.Connect(addr)
		util.Log("Mp2SessionManager.Mp2ConnectForPull(): Peer Addr=%s, SessionID=%d", addr, session.SessionID)

		if p.mp2sessionMap[mp2ID] == nil {
			// Create a mp2session using the given peer session
			mp2sess, err = CreateMp2Session(session, p)
			if err != nil {
				panic(err)
			}

			util.Log("Mp2SessionManager.Mp2ConnectForPull(): Create Mp2Session! (Mp2SessionID=%d)", mp2ID)

			mp2sess.key = mp2ID

			p.blockNumChannel[mp2ID] = make(chan uint32, 1)
			p.blockInfoChannel[mp2ID] = make(chan *BlockInfoPacket, 1)
			p.blockFinishFlag[mp2ID] = make(chan bool, 1)

			p.mp2sessionMap[mp2ID] = mp2sess
		} else {
			p.mp2sessionMap[mp2ID].AddSession(session)
		}

		p.mp2sessionMap[mp2ID].HandlerStart(session.SessionID)
	}

	p.mp2IdForPull = mp2ID
	p.peerPushMode = false

	// Send Control Packet to signal mode change
	p.Mp2WriteControl([]byte("Pull Mode"), p.mp2IdForPull)

	// To close connection with parent node
	p.blockFinishFlag[p.mp2IdFromParent] <- true

	// Send Block Fin Packet to force disconnection
	for sessionID := range p.mp2sessionMap[p.mp2IdFromParent].sessionMap {
		p.mp2sessionMap[p.mp2IdFromParent].sendBlockFinPacket(sessionID)
	}

	// Send Block Find Packet
	p.Mp2FindBlock(blockNumber)
}

// Mp2Listen() function manages mp2bs connection and prepares to receive data.
func (p *Mp2SessionManager) Mp2Listen() {

	buf := make([]byte, 1024)

	for {
		// Wait to receive mp2session ID
		mp2Id := p.Mp2Accept()

		p.Mp2ReadControl(buf, mp2Id)

		if strings.Contains(string(buf), "Push Mode") {
			util.Log("Mp2SessionManager.Mp2Listen(): Push Mode (Mp2SessionID=%d)", mp2Id)

			p.mp2IdFromParent = mp2Id

			// Waiting for receiving Block Info Packet at non-leaf node
			go p.Mp2ReceiveBlockInfo(p.mp2IdFromParent)
		} else if strings.Contains(string(buf), "Pull Mode") {
			util.Log("Mp2SessionManager.Mp2Listen(): Pull Mode (Mp2SessionID=%d)", mp2Id)

			p.mp2IdForPull = mp2Id

			// Waiting for receiving Block Find Packet for pull mode
			blockNumber := p.Mp2ReceiveBlockFind(p.mp2IdForPull)

			// Check if mp2session manager has a block
			_, exist := p.blockList[blockNumber]
			// If has
			if exist {
				// Send Block Info Packet
				p.Mp2SendBlockInfo(p.mp2IdForPull, blockNumber, p.blockList[blockNumber])
			}
		}
	}
}

// Mp2Accept() function returns mp2session ID.
// It should be called in a loop.
func (p *Mp2SessionManager) Mp2Accept() uint32 {

	// Accept a peer session
	session := p.sessionManager.Accept()
	util.Log("Mp2SessionManager.Mp2Accept(): SessionID=%d", session.SessionID)

	// Create a mp2session using the given peer session
	mp2sess, err := CreateMp2Session(session, p)
	if err != nil {
		panic(err)
	}

	// Make a random ID for mp2sessionMap key
	mp2ID := rand.Uint32()
	p.mp2sessionMap[mp2ID] = mp2sess

	util.Log("Mp2SessionManager.Mp2Accept(): Create Mp2Session! (Mp2SessionID=%d)", mp2ID)

	p.blockNumChannel[mp2ID] = make(chan uint32, 1)
	p.blockInfoChannel[mp2ID] = make(chan *BlockInfoPacket, 1)
	p.blockFinishFlag[mp2ID] = make(chan bool, 1)

	mp2sess.key = mp2ID

	// Start mp2session handler
	p.mp2sessionMap[mp2ID].HandlerStart(session.SessionID)

	return mp2ID
}

// Mp2FindBlock() function sends Block Find Packet to find the block to receive.
//
// It will be called by nodes that have not received the block from their parent node.
func (p *Mp2SessionManager) Mp2FindBlock(blockNumber uint32) {
	util.Log("Mp2SessionManager.Mp2FindBlock(): BlockNumber=%d", blockNumber)

	// Send Block Find Packet
	for sessionID := range p.mp2sessionMap[p.mp2IdForPull].sessionMap {
		p.mp2sessionMap[p.mp2IdForPull].FindBlock(sessionID, blockNumber)
	}
}

// Mp2ReceiveBlockFind() function receives block number other nodes want to receive.
func (p *Mp2SessionManager) Mp2ReceiveBlockFind(mp2ID uint32) uint32 {

	// Wait until Block Find Packet is arrived
	blockNumber := <-p.blockNumChannel[mp2ID]

	util.Log("Mp2SessionManager.Mp2ReceiveBlockFind(): BlockNumber=%d ", blockNumber)

	return blockNumber
}

// Mp2ReceiveBlockInfo() function receives Block Info Packet.
func (p *Mp2SessionManager) Mp2ReceiveBlockInfo(mp2ID uint32) {
	var packet *BlockInfoPacket

	for {
		// Wait until Block Info Packet is arrived
		util.Log("Mp2SessionManager.Mp2ReceiveBlockInfo(): Wait Block Info Packet (Mp2SessionID=%d)", mp2ID)
		packet = <-p.blockInfoChannel[mp2ID]

		// Set block information based on the received packet
		p.blockList[packet.BlockNumber] = packet.BlockSize
		p.receivedBlockChannel <- packet.BlockNumber

		util.Log("Mp2SessionManager.Mp2ReceiveBlockInfo(): BlockNumber=%d, BlockSize=%d ", packet.BlockNumber, packet.BlockSize)

		// Send to children if non-leaf node (PUSH mode)
		if p.mp2IdToChildren != 0 {
			<-p.nodeInfoChan

			// Send Block Info Packet
			p.Mp2SendBlockInfo(p.mp2IdToChildren, packet.BlockNumber, packet.BlockSize)

			// FIXME: may cause bug
			// To skip waiting for node info packet
			p.nodeInfoChan <- true
		}
	}
}

// Mp2SendBlock() function sends the requested block.
// We assume that the requested block is "blockName" file.
//
// It will be called by the root peer trying to propagate the block.
func (p *Mp2SessionManager) Mp2SendBlock(blockName string, blockNumber uint32) {

	// Get block size
	stat, err := os.Stat(blockName)
	if err != nil {
		panic(err)
	}

	// Store block information
	p.blockList[blockNumber] = uint32(stat.Size())

	// Wait for Node Info Packet to arrive
	<-p.nodeInfoChan

	// Send Block Info Packet
	p.Mp2SendBlockInfo(p.mp2IdToChildren, blockNumber, p.blockList[blockNumber])

	// Open block and write to buffer
	fi, err := os.Open(blockName)
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 1024)
	total := 0
	for {
		// Read data from block (file)
		n, err := fi.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		} else if err == io.EOF {
			break
		}

		total = total + n

		// Write data into mp2session's buffer
		p.AddSendBuffer(buf[:n])
	}
	util.Log("Mp2SessionManager.Mp2SendBlock(): Write %d bytes from block file into Block Buffer! \n", total)

	// FIXME: may cause bug
	// To skip waiting for node info packet
	p.nodeInfoChan <- true
}

// Mp2SendBlockInfo() function checks whether p.blockList contains the requested block
// and sends Block Info Packet.
func (p *Mp2SessionManager) Mp2SendBlockInfo(mp2ID uint32, blockNumber uint32, blockSize uint32) {

	status := config.PEER_HAS_NO_BLOCK

	for _, s := range p.blockList {
		if blockNumber == s {
			status = config.PEER_HAS_BLOCK
			break
		}
	}

	p.mp2sessionMap[mp2ID].BlockInfo(blockNumber, blockSize, uint32(status))
}

// Mp2SendNodeInfoForAnchor() function sends Node Info Packet to top node.
//
// It will be called by anchor node for block propagation test.
func (p *Mp2SessionManager) Mp2SendNodeInfoForAnchor() {

	// Send Control Packet so childs are ready to receive data
	p.Mp2WriteControl([]byte("Push Mode"), p.mp2IdToChildren)

	for sessionID := range p.mp2sessionMap[p.mp2IdToChildren].sessionMap {
		p.mp2sessionMap[p.mp2IdToChildren].SendNodeInfo(sessionID)
	}

}

// Mp2SendNodeInfo() function sends Node Info Packet to child nodes.
func (p *Mp2SessionManager) Mp2SendNodeInfo(nodeInfos []NodeInfo) {

	if len(p.childAddr) != 0 {
		// Connect to childs
		p.mp2IdToChildren = p.Mp2ConnectToChildren()

		// Send Control Packet so childs are ready to receive data
		p.Mp2WriteControl([]byte("Push Mode"), p.mp2IdToChildren)

		for sessionID := range p.mp2sessionMap[p.mp2IdToChildren].sessionMap {
			// To check whether Node Info Ack Packet has been received
			p.nodeInfoFlag[sessionID] = false

			util.Log("Mp2SessionManager.Mp2SendNodeInfo(): SessionID=%d, p.nodeInfoFlag=%t", sessionID, p.nodeInfoFlag[sessionID])

			// Send Node Info Packet
			p.mp2sessionMap[p.mp2IdToChildren].sendNodeInfoPacket(sessionID, nodeInfos)

		}
	}

}

// Mp2ReadBlock() function reads the requested block and stores it in the given byte slice.
func (p *Mp2SessionManager) Mp2ReadBlock(buf []byte, blockNumber uint32) (int, error) {

	readLen := 0
	complete := false

	// TODO: infinit-loop
	for p.recvBuffer[blockNumber].GetLength() == 0 && !p.goodbye {
	}

	if p.recvBuffer[blockNumber].GetLength() > 0 {
		p.mutexForBuffer.Lock()
		segment := p.recvBuffer[blockNumber].ReadSegment()

		// TODO: inefficient process
		for !segment.hasDataToRead() {
			p.mutexForTimeout.Lock()
			if p.timeoutFlag[blockNumber] {
				p.mutexForBuffer.Unlock()
				// Wait until new segment is created
				for p.recvBuffer[blockNumber].buffer == nil {
				}
				p.mutexForBuffer.Lock()
				// New segment (PULL mode)
				segment = p.recvBuffer[blockNumber].ReadSegment()
			}
			p.mutexForTimeout.Unlock()
		}

		// Read data from the segment
		readLen, complete = segment.PopData(buf)

		// If read all data from the segment..
		if complete {
			// Last segment (finish)
			if segment.Last {
				p.goodbye = true
			}

			util.Log("Mp2SessionManager.Mp2ReadBlock(): Remove segment!!")

			// Remove the segment from recvBuffer
			p.recvBuffer[blockNumber].RemoveSegment()
		}
		p.mutexForBuffer.Unlock()
	}

	p.recvBuffer[blockNumber].readBytes += uint32(readLen)
	util.Log("Mp2SessionManager.Mp2ReadBlock(): Read len=%d", readLen)

	if p.peerPushMode {
		p.peerModeStatus = fmt.Sprintf("Push Mode: Download from parent!")
	} else {
		p.peerModeStatus = fmt.Sprintf("Pull Mode: Download from pull node(s)!")
	}

	return readLen, nil
}

// Mp2ReadControl() function is read API for Control Packet.
func (p *Mp2SessionManager) Mp2ReadControl(b []byte, mp2ID uint32) (int, error) {
	readLen := p.mp2sessionMap[mp2ID].ReadControl(b)

	util.Log("Mp2SessionManager.Mp2ReadControl(): Read len=%d", readLen)

	return readLen, nil
}

// Mp2WriteControl() function is write API for Control Packet.
func (p *Mp2SessionManager) Mp2WriteControl(b []byte, mp2ID uint32) (int, error) {
	p.mp2sessionMap[mp2ID].WriteControl(b)

	util.Log("Mp2SessionManager.Mp2WriteControl(): Write len=%d", len(b))

	return 0, nil
}

// Mp2SetNodInfo() function sets all node informations received from parent node.
func (p *Mp2SessionManager) Mp2SetNodeInfo(nodeInfos []NodeInfo) {
	myIdx := 0

	// 1. Find my ip address
	for idx, nodeInfo := range nodeInfos {
		for _, nicInfo := range config.Conf.NIC_ADDRS {
			ip := strings.Split(nicInfo, ":")          // "127.0.0.1:5000 --> 127.0.0.1 // 5000"
			port, _ := strconv.ParseInt(ip[1], 10, 16) // Convert string to uint16

			if util.Int2ip(nodeInfo.NodeIP) == ip[0] {
				if nodeInfo.Port == uint16(port) {
					util.Log("Mp2SessionManager.Mp2SetNodeInfo(): Find my IP=%s:%d", ip[0], nodeInfo.Port)
					myIdx = idx

					break
				}
			}
		}
	}

	// 2. Append child addresses to childAddr[] based on child offest
	if nodeInfos[myIdx].NumOfChilds != 0 {
		firstChildIdx := nodeInfos[myIdx].OffsetOfChild
		endChildIdx := firstChildIdx + nodeInfos[myIdx].NumOfChilds

		// Set child nodes infromation
		for i := firstChildIdx; i < endChildIdx; i++ {
			ipAddr := util.Int2ip(nodeInfos[i].NodeIP) + ":" + strconv.FormatInt(int64(nodeInfos[i].Port), 10)
			p.childAddr = append(p.childAddr, ipAddr)

			util.Log("Mp2SessionManager.Mp2SetNodeInfo(): Add child address=%s", ipAddr)
		}
	}

	// 3. Append all addresses to allAddr[]
	for i := 0; i < len(nodeInfos); i++ {
		if i == myIdx {
			continue
		}

		ipAddr := util.Int2ip(nodeInfos[i].NodeIP) + ":" + strconv.FormatInt(int64(nodeInfos[i].Port), 10)
		p.allAddr = append(p.allAddr, ipAddr)

		util.Log("Mp2SessionManager.Mp2SetNodeInfo(): Add all address=%s", ipAddr)
	}
}

// FIXME: we should improve close process.
//
//	Mp2Close() function terminates mp2session connection.
//	There are three cases:
//		1. parent connection
//		2. child connection
//		3. pull connection
func (p *Mp2SessionManager) Mp2Close() {

	// 1) Close connection with parent node (PUSH mode - receiver)
	if p.blockFinishFlag[p.mp2IdFromParent] != nil {
		// To close parent connection when it is a top node
		if p.peerId == 0 {
			p.mp2sessionMap[p.mp2IdFromParent].Close()
		} else {
			<-p.blockFinishFlag[p.mp2IdFromParent]

			p.mp2sessionMap[p.mp2IdFromParent].Close()
		}

		util.Log("Mp2SessionManager.Mp2Close(): Close Mp2Session for parent (Mp2SessionID=%d)", p.mp2IdFromParent)
	}

	// 2) Close connection with child nodes (PUSH mode - sender)
	if p.blockFinishFlag[p.mp2IdToChildren] != nil {
		<-p.blockFinishFlag[p.mp2IdToChildren]

		p.mp2sessionMap[p.mp2IdToChildren].Close()

		util.Log("Mp2SessionManager.Mp2Close(): Close Mp2Session for child (Mp2SessionID=%d)", p.mp2IdToChildren)
	}

	// 3. Close connection for pull mdoe (PULL mode - receiver && sender)
	if p.blockFinishFlag[p.mp2IdForPull] != nil {
		<-p.blockFinishFlag[p.mp2IdForPull]

		p.mp2sessionMap[p.mp2IdForPull].Close()

		util.Log("Mp2SessionManager.Mp2Close(): Close Mp2Session for pull mode (Mp2SessionID=%d)", p.mp2IdForPull)
	}

}

/*
	not yet developed

func (p *Mp2SessionManager) Mp2BypassConnect(addr string) {
	// // Connect to Master listener
	// udpAddr, err := net.ResolveUDPAddr("udp", addr)
	// if err != nil {
	// 	panic(err)
	// }

	// // TODO bind my IP?
	// ip4 := net.ParseIP("127.0.0.1").To4()
	// udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: ip4, Port: 0})
	// if err != nil {
	// 	panic(err)
	// }

	// // TLS configuration
	// tlsConf := &tls.Config{
	// 	InsecureSkipVerify: true,
	// 	NextProtos:         []string{"socket-programming"},
	// }

	// // QUIC Dial
	// quicSess, err := quic.Dial(udpConn, udpAddr, addr, tlsConf, nil)
	// if err != nil {
	// 	panic(err)
	// }

	// // QUIC OpenStreamSync
	// quicStream, err := quicSess.OpenStreamSync(context.Background())
	// if err != nil {
	// 	panic(err)
	// }

}

func (p *Mp2SessionManager) Mp2BypassRead(buf []byte) {
	// quicStream.Read(buf)
}

func (p *Mp2SessionManager) Mp2BypassWrite(buf []byte) {
	// quicStream.Write(buf)
}
*/

// AddRecvBuffer() function adds the given segment to recvBuffer[] based on block number.
func (p *Mp2SessionManager) AddRecvBuffer(segment *Segment) {
	p.mutexForBuffer.Lock()

	blockNumber := segment.BlockNumber

	// Push segment into block buffer
	p.recvBuffer[blockNumber].PushSegment(segment)

	p.mutexForBuffer.Unlock()
}

// AddSendBuffer() function adds the given data to sendBuffer[].
func (p *Mp2SessionManager) AddSendBuffer(buf []byte) {
	p.sendBuffer = append(p.sendBuffer, buf...)
}

// Mp2GetBlockInfo() function returns block information. (block number && block size)
func (p *Mp2SessionManager) Mp2GetBlockInfo() (uint32, uint32) {

	blockNumber := <-p.receivedBlockChannel

	return blockNumber, p.blockList[blockNumber]

}

// Mp2GetPeerStatus() function returns mode status. (PUSH or PULL)
func (p *Mp2SessionManager) Mp2GetPeerStatus() string {
	return p.peerModeStatus
}

// Mp2Test() function supports mp2bs performance(throughput) test.
//
//	 It receives two parameters:
//		rate = estimated throughput by user (unit: Mbit)
//		time = total execution time (unit: sec)
func (p *Mp2SessionManager) Mp2Test(rate float64, time float64) {

	// Randomly assigned value for test(any value available)
	blockNumber := uint32(100)

	// Calculate proper block size
	totalSize := (rate * (1000 * 1000) / 8) * time

	fmt.Printf("TEST SETUP (Total Size=%.2f, Estimated execution time=%.2f)\n", totalSize, time)

	// Store block information
	p.blockList[blockNumber] = uint32(totalSize)

	// Wait for Node Info Packet to arrive
	<-p.nodeInfoChan

	// Send Block Info Packet
	p.Mp2SendBlockInfo(p.mp2IdToChildren, blockNumber, p.blockList[blockNumber])

	buf := make([]byte, 1024)
	n := 0
	sentSize := 0

	// Read only once
	n, _ = rand.Read(buf)

	for sentSize != int(totalSize) {

		// Remaining data size <= n
		if int(totalSize)-sentSize <= n {
			n = int(totalSize) - sentSize
		}

		sentSize = sentSize + n

		// Write data into mp2session's buffer
		p.AddSendBuffer(buf[:n])
	}

	// FIXME: may cause bug
	// To skip waiting for Node Info Packet
	p.nodeInfoChan <- true
}
