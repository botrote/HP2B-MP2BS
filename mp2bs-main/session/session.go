package session

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"mp2bs/config"
	"mp2bs/util"
	"net"
	"strings"
	"sync"
	"time"

	quic "github.com/quic-go/quic-go"
)

// For multipath session
type Session struct {
	mutex             []sync.Mutex
	SessionID         uint32
	numPath           int
	streamList        []quic.Stream
	listenAddrList    []string
	connectedAddrList []string
	sequenceNumber    uint32
	sentBytes         []uint32
	recvBytes         []uint32
	throughput        []float64
	scheduler         *SessionScheduler
	recvBuffer        *RecvBuffer
	sendBuffer        [][]*DataPacket

	goodbye bool
	checker chan struct{}

	timeline time.Time
}

func CreateSession(sessionID uint32, addrList []string) *Session {
	s := Session{
		mutex:             make([]sync.Mutex, 2),
		SessionID:         sessionID,
		numPath:           0,
		streamList:        make([]quic.Stream, 0),
		listenAddrList:    addrList,
		connectedAddrList: make([]string, 0),
		sequenceNumber:    0,
		sentBytes:         make([]uint32, 0),
		recvBytes:         make([]uint32, 0),
		throughput:        make([]float64, 0),
		scheduler:         CreateSessionScheduler(SCHED_USER_WRR),
		recvBuffer:        CreateRecvBuffer(),
		sendBuffer:        make([][]*DataPacket, 0),
		goodbye:           false,
		checker:           make(chan struct{}, 1),
		timeline:          time.Now(),
	}

	return &s
}

func (s *Session) Connect(addr string, i int) {
	// Connect to Master listener
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}

	// To get address without port number
	slice := strings.Split(s.listenAddrList[i], ":")

	// Bind my IP
	ip4 := net.ParseIP(slice[0]).To4()
	// port, _ := strconv.Atoi(slice[1])
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: ip4, Port: 0})
	if err != nil {
		panic(err)
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"mp2bs"},
		MaxVersion:         tls.VersionTLS13,
		MinVersion:         tls.VersionTLS10,
	}

	// QUIC Dial
	quicSession, err := quic.Dial(udpConn, udpAddr, addr, tlsConf, nil)
	if err != nil {
		panic(err)
	}

	// QUIC OpenStreamSync
	quicStream, err := quicSession.OpenStreamSync(context.Background())
	if err != nil {
		panic(err)
	}

	util.Log("Session.Connect(): Connect to %s (from %s)", addr, slice[0])

	// fmt.Printf("Session.Connect(): Connect to %s (from %s) - %.2f\n", addr, s.listenAddrList[i], time.Since(s.timeline).Seconds())

	// Add a created session into session map
	pathID := s.AddStream(quicStream, quicSession.RemoteAddr().String(), false)

	// Send a Hello Packet
	s.sendHelloPacket(pathID)

	// Receive a Hello ACK Packet
	s.receiveHelloAckPacket(pathID)

	// Start peer session receiver
	s.StartReceiver(pathID)
}

func (s *Session) AddStream(stream quic.Stream, connectedAddr string, accepted bool) int {
	s.streamList = append(s.streamList, stream)
	s.connectedAddrList = append(s.connectedAddrList, connectedAddr)
	s.numPath++
	s.sentBytes = append(s.sentBytes, 0)
	s.recvBytes = append(s.recvBytes, 0)
	s.throughput = append(s.throughput, 0)

	s.sendBuffer = append(s.sendBuffer, make([]*DataPacket, 0))
	go s.dataPacketTransmitter(s.numPath - 1)

	if accepted {
		s.scheduler.SetNumPath(s.numPath)
	}

	return (s.numPath - 1)
}

// Receive a Hello ACK Packet
func (s *Session) receiveHelloAckPacket(pathID int) {
	// Get stream instance
	stream := s.streamList[pathID]

	buf := make([]byte, config.Conf.PACKET_SIZE)

	// Read packet type and length
	_, err := stream.Read(buf[:3])
	if err != nil {
		panic(err)
	}

	r := bytes.NewReader(buf[:3])
	packetType, _ := r.ReadByte()
	packetLength, _ := util.ReadUint16(r)

	// Receive remaining data
	_, err = stream.Read(buf[3:packetLength]) // Read after field of packet length
	if err != nil {
		panic(err)
	}

	// Parse packet
	reader := bytes.NewReader(buf)
	if packetType == config.Conf.HELLO_ACK_PACKET {
		packet, err := ParseHelloAckPacket(reader)
		if err != nil {
			panic(err)
		}
		s.handleHelloAckPacket(packet)
	} else {
		panic(fmt.Sprintf("Session.receiveHelloAckPacket(): Unknown packet type (%d)", packetType))
	}
}

func (s *Session) StartReceiver(pathID int) {
	// Start receiver
	go s.receiver(pathID)
}

// TODO: DeleteStream()

// Packet receiver
func (s *Session) receiver(pathID int) {
	// Get stream instance
	stream := s.streamList[pathID]
	total := 0
	count := 0
	startTime := time.Now()

	var len int

	for {
		buf := make([]byte, config.Conf.PACKET_SIZE)

		// Receive packet type and length
		_, err := stream.Read(buf[:3])
		if err != nil {
			if err == io.EOF {
				// TODO: just break?
				break
			} else {
				panic(err)
			}
		}

		r := bytes.NewReader(buf[:3])
		packetType, _ := r.ReadByte()         // 1 byte
		packetLength, _ := util.ReadUint16(r) // 2 bytes
		i := 3

		// Receive remaining data
		for i != int(packetLength) {
			len, err = stream.Read(buf[i:packetLength]) // Read after field of packet length
			if err != nil {
				if err == io.EOF {
					// TODO: just break?
					break
				} else {
					panic(err)
				}
			}
			i += len
		}

		count++
		total = total + i

		elapsedTime := time.Since(startTime).Seconds()
		s.throughput[pathID] = (float64(total) * 8.0) / float64(elapsedTime) / (1000 * 1000)

		// TODO: not necessary
		if count%1000 == 0 {
			util.Log("Session.receiver(): PathID=%d, Throughput=%.2fMbps, Total=%d\n", pathID, s.throughput[pathID], total)
		}

		reader := bytes.NewReader(buf)

		// Packet Handling
		switch packetType {
		// Hello Packet or Hello ACK Packet
		case config.Conf.HELLO_PACKET:
		case config.Conf.HELLO_ACK_PACKET:
			// Error case since Hello Packet is received when the session created
			panic(fmt.Sprintf("Session.receiver(): Error! PathID=%d, Packet Type=%d !!!!!!!!!!!!!!!!!!!!!\n", pathID, packetType))

		// Data Packet
		case config.Conf.DATA_PACKET:
			packet, err := ParseDataPacket(reader)
			if err != nil {
				panic(err)
			}

			s.recvBytes[pathID] += uint32(packet.Length - DATA_PACKET_HEADER_LEN)

			s.handleDataPacket(packet)

		// Goodbye Packet
		// TODO: remove or modify
		case config.Conf.GOODBYE_PACKET:
			packet, err := ParseGoodbyePacket(reader)
			if err != nil {
				panic(err)
			}
			s.handleGoodbyePacket(packet)

		default:
			panic(fmt.Sprintf("Unknown packet type: %d \n", packetType))
		}
	}
}

// Send a Hello Packet
func (s *Session) sendHelloPacket(pathID int) {
	util.Log("Session.sendHelloPacket(): SessionID=%d", s.SessionID)

	// Create a Hello Packet and convert it into byte[].
	// Session ID of first hello packet is 0.
	// After first hello packet, session ID is greater than 0. (assigned by server)
	packet := CreateHelloPacket(s.SessionID, byte(len(s.listenAddrList)))
	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(b.Bytes(), pathID)
}

// Send a Hello ACK Packet
func (s *Session) sendHelloAckPacket(pathID int) {
	util.Log("Session.sendHelloAckPacket(): SessionID=%d", s.SessionID)

	nicInfos := s.getNicInfo()

	// Create a Hello ACK Packet and convert it into byte[].
	packet := CreateHelloAckPacket(s.SessionID, nicInfos)
	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(b.Bytes(), pathID)
}

// Send a Data Packet
func (s *Session) sendDataPacket(payload []byte, pathID int) {
	util.Log("Session.sendDataPacket(): SessionID=%d, PathID=%d, Len.Payload=%d", s.SessionID, pathID, len(payload))

	// Create a Data Packet and convert it into byte[].
	packet := CreateDataPacket(s.SessionID, pathID, s.sequenceNumber, payload)

	s.mutex[pathID].Lock()
	s.sendBuffer[pathID] = append(s.sendBuffer[pathID], packet)
	s.mutex[pathID].Unlock()
}

// Send a Goodbye Packet
func (s *Session) sendGoodbyePacket(pathID int) {
	util.Log("Session.sendGoodbyePacket(): SessionID=%d, PathID=%d", s.SessionID, pathID)

	packet := CreateGoodbyePacket(s.SessionID)
	b := &bytes.Buffer{}
	packet.Write(b)

	// Send bytes of packet
	s.sendPacket(b.Bytes(), pathID)
}

func (s *Session) sendPacket(packet []byte, pathID int) {
	stream := s.streamList[pathID]
	_, err := stream.Write(packet)
	if err != nil {
		log.Println(err)
	}
}

func (s *Session) dataPacketTransmitter(pathID int) {

	var packet *DataPacket

	for {
		// Get packet from sendBuffer
		for {
			if len(s.sendBuffer[pathID]) > 0 {
				break
			}
		}

		// TODO: parallel writing
		// There is a problem with excessive stack of sendBuffer. Also, Lock time is to long (priority problem?)
		// First, we need to check if it is a problem with mutex. It may be a QIUC performance problem.
		// How to check it?

		s.mutex[pathID].Lock()

		packet = s.sendBuffer[pathID][0]
		s.sendBuffer[pathID] = s.sendBuffer[pathID][1:]

		s.mutex[pathID].Unlock()

		util.Log("Session.dataPacketTransmitter(): SessionID=%d, PathID=%d, PacketSeqNumber=%d\n",
			s.SessionID, pathID, packet.SeqNumber)

		b := &bytes.Buffer{}
		packet.Write(b)

		// Send bytes of packet
		s.sendPacket(b.Bytes(), pathID)
	}
}

// Handle a Hello ACK Packetls
func (s *Session) handleHelloAckPacket(packet *HelloAckPacket) {
	util.Log("Session.handleHelloAckPacket(): SessionID=%d", packet.SessionID)

	// Set session ID assigned by server
	if s.SessionID == 0 {
		s.SessionID = packet.SessionID
	}

	// Set numPath for scheduler -> scheduler begins to consider an added path
	s.scheduler.SetNumPath(s.numPath)

	for i, nicInfo := range packet.NicInfos {
		nicAddr := string(nicInfo.Addr)
		util.Log("Session.handleHelloAckPacket(): NicInfo[%d]=%s", i, nicAddr)
	}

	// TODO: Hardcorded mapping
	if len(s.connectedAddrList) == 1 {
		if packet.NumPath == 2 && len(s.listenAddrList) == 1 {
			nicAddr := string(packet.NicInfos[1].Addr)
			s.Connect(nicAddr, 0)
		} else if packet.NumPath == 1 && len(s.listenAddrList) == 2 {
			nicAddr := string(packet.NicInfos[0].Addr)
			s.Connect(nicAddr, 1)
		} else if packet.NumPath == 2 && len(s.listenAddrList) == 2 {
			nicAddr := string(packet.NicInfos[1].Addr)
			s.Connect(nicAddr, 1)
		}
	}
}

// Handle a Data Packet
func (s *Session) handleDataPacket(packet *DataPacket) {
	s.recvBuffer.PushPacket(packet)
}

// Handle a Goodbye Packet
func (s *Session) handleGoodbyePacket(packet *GoodbyePacket) {
	// To terminate receiver go routine
	util.Log("Session.handleGoodbyePacket(): SessionID=%d", packet.SessionID)

	s.goodbye = true
}

// Read data
func (s *Session) Read(buf []byte) (int, error) {

Loop:
	for {
		select {
		case <-s.checker:
			// Check whether recvBuffer is empty
			if !s.recvBuffer.IsEmpty() || s.goodbye {
				break Loop
			}
		default:
			if len(s.checker) == 0 {
				s.checker <- struct{}{}
			}
		}

	}

	readLen := s.recvBuffer.Read(buf)
	// util.Log("Session.Read(): Read from the recvBuffer=%d", readLen)

	return readLen, nil
}

// Read data
func (s *Session) ReadWithSize(buf []byte, size int) (int, error) {

Loop:
	for {
		select {
		case <-s.checker:
			// Check whether recvBuffer is empty
			if s.recvBuffer.GetLength() >= size || s.goodbye {
				break Loop
			}
		default:
			if len(s.checker) == 0 {
				s.checker <- struct{}{}
			}
		}

	}

	readLen := s.recvBuffer.Read(buf)
	// util.Log("Session.ReadWithSize(): Read from the recvBuffer=%d", readLen)

	return readLen, nil
}

// Write data
func (s *Session) Write(buf []byte, pType byte) (int, error) {
	start, end := 0, 0
	pathID := 0
	total := 0

	for start < len(buf) {
		// FIXME: move this part to dataPacketTransmitter()
		// Compute remaining time
		if pType == 14 {
			s.scheduler.monitor.computeRemainingTime(uint32(len(buf) - end))
		}
		// Determine the range of payload
		if start+DATA_PACKET_PAYLOAD_SIZE < len(buf) {
			end = start + DATA_PACKET_PAYLOAD_SIZE
		} else {
			end = len(buf)
		}

		payloadSize := uint32(end - start)
		total += int(payloadSize)

		// Scheduling
		pathID = s.scheduler.Scheduling(payloadSize)

		// Send Data Packet
		s.sendDataPacket(buf[start:end], pathID)
		s.sentBytes[pathID] += payloadSize
		s.sequenceNumber++

		start = end

		// FIXME: move this part to dataPacketTransmitter()
		// Compute throughput
		if pType == 14 {
			s.scheduler.monitor.computeThroughput(payloadSize, time.Now())
		}
	}

	return total, nil
}

// TODO: how to automatically get my NIC information?
// We are currently using config file to get NIC infomration.
func (s *Session) getNicInfo() []NicInfo {
	nicInfos := make([]NicInfo, len(s.listenAddrList))
	for i := 0; i < len(s.listenAddrList); i++ {
		nicInfos[i].Type = 0
		nicInfos[i].AddrLen = byte(len(s.listenAddrList[i]))
		nicInfos[i].Addr = []byte(s.listenAddrList[i])
	}
	return nicInfos
}

// For testing multipeer using docker
/*
	1. peer0
		- eth0: 172.18.0.2:4242 <---> 192.168.10.3:4242
   		- eth1: 172.19.0.2:4243 <---> 203.252.112.32:4243
	2. peer1
 		- eth0: 172.18.0.3:4343 <---> 192.168.10.3:4343
   		- eth1: 172.19.0.3:4344 <---> 203.252.112.32:4344
	3. peer2
 		- eth0: 172.18.0.4:4444 <---> 192.168.10.3:4444
   		- eth1: 172.19.0.4:4445 <---> 203.252.112.32:4445
	4. peer3
	 	- eth0: 172.18.0.5:4545 <---> 192.168.10.3:4545
   		- eth1: 172.19.0.5:4546 <---> 203.252.112.32:4546
	5. peer4
	 	- eth0: 172.18.0.6:4646 <---> 192.168.10.3:4646
   		- eth1: 172.19.0.6:4647 <---> 203.252.112.32:4647
	6. anchor
   		-eth0: 172.18.0.2:4251 <---> 192.168.10.2:4251
   		-eth1: 172.19.0.2:4252 <---> 203.252.112.31:4252
*/

/*
func (s *Session) getNicInfo() []NicInfo {
	nicInfos := make([]NicInfo, len(s.listenAddrList))
	for i := 0; i < len(s.listenAddrList); i++ {
		nicInfos[i].Type = 0
		if i == 0 {
			nicInfos[i].AddrLen = byte(len("192.168.10.2:4251"))
			nicInfos[i].Addr = []byte("192.168.10.2:4251")
		} else {
			nicInfos[i].AddrLen = byte(len("203.252.112.31:4252"))
			nicInfos[i].Addr = []byte("203.252.112.31:4252")
		}
	}
	return nicInfos
}
*/

// FIXME: we should improve close process
func (s *Session) Close() {

	time.Sleep(time.Duration(config.Conf.CLOSE_TIMEOUT_PERIOD) * time.Millisecond)

	for idx, stream := range s.streamList {
		stream.Close()

		util.Log("Session.Close(): Close stream=%d", idx)
	}
}
