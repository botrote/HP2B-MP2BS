package session

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"mp2bs/config"
	"mp2bs/util"
	"sync"

	quic "github.com/quic-go/quic-go"
)

// Session Manager
type SessionManager struct {
	mutex          sync.Mutex
	numPath        int
	listenerList   []quic.Listener
	listenAddrList []string
	sessionMap     map[uint32]*Session
	numPathChan    chan byte
	sessionChan    chan *Session
	listenFlag     bool
}

func CreateSessionManager(addrList []string) *SessionManager {
	// Create a SessionManager
	m := SessionManager{
		numPath:        len(addrList),
		listenerList:   make([]quic.Listener, len(addrList)),
		listenAddrList: addrList,
		sessionMap:     make(map[uint32]*Session),
		sessionChan:    make(chan *Session),
		listenFlag:     false,
	}

	m.listen()

	return &m
}

func (m *SessionManager) listen() {
	var err error

	// TODO: QUIC configuration for enhanced QUIC
	config := quic.Config{}

	// QUIC ListenAddr
	for i, addr := range m.listenAddrList {
		util.Log("SessionManger.listen(): ListenAddr[%d]=%s", i, addr)
		m.listenerList[i], err = quic.ListenAddr(addr, util.GenerateTLSConfig(), &config)
		if err != nil {
			panic(err)
		}
	}
}

func (m *SessionManager) Accept() *Session {
	var numberOfWaits int
	var sess *Session
	m.numPathChan = make(chan byte)

	// Start go routines for all listen addresses
	if !m.listenFlag {
		for i := 0; i < m.numPath; i++ {
			go m.accept(i, context.Background())
		}
		m.listenFlag = true
	}

	// Remote number of path
	remoteNumPath := <-m.numPathChan

	// Flush numPath channel to block signal
	m.numPathChan = nil

	// Set remote number of path
	if m.numPath > int(remoteNumPath) {
		numberOfWaits = m.numPath
	} else {
		numberOfWaits = int(remoteNumPath)
	}

	// Block until multipath connection is complete
	for i := 0; i < numberOfWaits; i++ {
		sess = <-m.sessionChan
	}

	// Add a new session into session map
	m.sessionMap[sess.SessionID] = sess

	return sess
}

func (m *SessionManager) accept(pathID int, ctx context.Context) {

	for {
		// QUIC Accept
		quicSession, err := m.listenerList[pathID].Accept(ctx)
		if err != nil {
			panic(err)
		}

		util.Log("SessionManger.accept(): PathID=%d, Accepted address=%s", pathID, quicSession.RemoteAddr().String())

		// QUIC AcceptStream
		quicStream, err := quicSession.AcceptStream(ctx)
		if err != nil {
			panic(err)
		}

		// Receive a Hello Packet
		sessionID := m.receiveHelloPacket(quicStream)

		m.mutex.Lock()
		var sess *Session
		if sessionID == 0 {
			// Assign a new session ID (first connection)
			sessionID = rand.Uint32()

			// Create a new session
			sess = CreateSession(sessionID, m.listenAddrList)
			m.sessionMap[sessionID] = sess
			util.Log("SessionManager.accept(): New session is created! (SessionID=%d)", sessionID)
		} else {
			// Get an existing session
			var exists bool
			sess, exists = m.sessionMap[sessionID]
			if !exists {
				panic(fmt.Sprintf("SessionManger.accept(): Received SessionID=%d is not 0 but not exists in the session map!", sessionID))
			} else {
				util.Log("SessionManager.accept(): New connection is added to existing session! (SessionID=%d)", sessionID)
			}
		}

		// Add a created session into session map
		newPathID := sess.AddStream(quicStream, quicSession.RemoteAddr().String(), true)
		m.mutex.Unlock()

		// Send a Hello ACK Packet
		sess.sendHelloAckPacket(newPathID)

		// Start a session receiver
		sess.StartReceiver(newPathID)

		// Send session instance for Accept()
		m.sessionChan <- sess
	}
}

// Receive a Hello Packet
func (s *SessionManager) receiveHelloPacket(quicStream quic.Stream) uint32 {
	buf := make([]byte, HELLO_PACKET_HEADER_LEN)

	// Read a Hello Packet from QUIC stream
	_, err := quicStream.Read(buf[:HELLO_PACKET_HEADER_LEN])
	if err != nil {
		panic(err)
	}

	// Parse packet type
	r := bytes.NewReader(buf[:HELLO_PACKET_HEADER_LEN])
	packetType, _ := r.ReadByte()

	// Parse packet
	if packetType == config.Conf.HELLO_PACKET {
		reader := bytes.NewReader(buf)
		packet, err := ParseHelloPacket(reader)
		if err != nil {
			panic(err)
		}
		util.Log("SessionManager.receiveHelloPacket(): SessionID=%d, numPath=%d", packet.SessionID, packet.NumPath)

		if s.numPathChan != nil {
			s.numPathChan <- packet.NumPath
		}

		return packet.SessionID
	} else {
		panic(fmt.Sprintf("SessionManager.receiveHelloPacket(): Unknown initial packet type (%d)", packetType))
	}
}

// Connect
func (m *SessionManager) Connect(addr string) *Session {
	// Create a session
	sess := CreateSession(0, m.listenAddrList)

	sess.Connect(addr, 0)

	return sess
}
