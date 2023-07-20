package session

import (
	config "mp2bs/config"
	util "mp2bs/util"
)

const (
	SCHED_USER_WRR = 1 // User-defined weight round robin
	SCHED_NET_WRR  = 2 // Newtwork condition based weight round robin
)

const REMAINING_BYTES_RESET_THRESH = DATA_PACKET_PAYLOAD_SIZE / 8

// Multipath session scheduler for packet transmission
type SessionScheduler struct {
	schedulerType  int
	numPath        int
	weight         []uint32
	remainingBytes []uint32
	currentPath    int
	usePath        int
	multipath      bool
	monitor        *SessionMonitor
}

func CreateSessionScheduler(schedType int) *SessionScheduler {
	c := SessionScheduler{
		schedulerType:  schedType,
		numPath:        0,
		remainingBytes: make([]uint32, 0),
		usePath:        0,
		multipath:      true,
		monitor:        CreateSessionMonitor(),
	}

	// Set weight
	if schedType == SCHED_USER_WRR {
		c.weight = make([]uint32, len(config.Conf.CONFIG_USER_WRR_WEIGHT))
		for i := 0; i < len(c.weight); i++ {
			c.weight[i] = config.Conf.CONFIG_USER_WRR_WEIGHT[i]
		}
	}

	go c.startMonitoring()

	return &c
}

func (c *SessionScheduler) SetNumPath(numPath int) {
	c.numPath = numPath
	c.usePath = c.numPath // default: use multipath, if it is available

	if c.usePath > 1 {
		// When the additional path is added..
		// Reset remaining bytes of current path
		c.remainingBytes[c.currentPath] = c.weight[c.currentPath] * DATA_PACKET_PAYLOAD_SIZE
	}

	// Change current path to new path and Set the remainig bytes
	c.currentPath = c.usePath - 1
	remainBytesOfNewPath := c.weight[c.currentPath] * DATA_PACKET_PAYLOAD_SIZE

	c.remainingBytes = append(c.remainingBytes, remainBytesOfNewPath)

	util.Log("SessionScheduler.SetNumPath(): numPath=%d, usePath=%d, len remainingBytes=%d", c.numPath, c.usePath, len(c.remainingBytes))
}

// Weighted RR
func (c *SessionScheduler) Scheduling(payloadSize uint32) int {
	pathID := 0

	switch c.schedulerType {
	case SCHED_USER_WRR:
		pathID = c.scheduling_user_wrr(payloadSize)

	case SCHED_NET_WRR:
		pathID = c.scheduling_net_wrr(payloadSize)

	default:
		pathID = c.scheduling_user_wrr(payloadSize)
	}

	return pathID
}

// User-defined weight round robin
func (c *SessionScheduler) scheduling_user_wrr(payloadSize uint32) int {
	selectedPath := c.currentPath

	// Update remaing bytes of selected path
	if c.remainingBytes[selectedPath] >= payloadSize {
		c.remainingBytes[selectedPath] -= payloadSize
	} else {
		c.remainingBytes[selectedPath] = 0
	}

	// util.Log("SessionScheduler.SchedulingUserWrr(): c.remainingBytes[%d]=%d", selectedPath, c.remainingBytes[selectedPath])

	// Reset remaining bytes of selected path and Change the current path to next path
	if c.remainingBytes[selectedPath] <= REMAINING_BYTES_RESET_THRESH {
		c.remainingBytes[selectedPath] = c.weight[selectedPath] * DATA_PACKET_PAYLOAD_SIZE
		c.currentPath = (c.currentPath + 1) % c.usePath
	}

	return selectedPath
}

// TODO: Need network information
func (c *SessionScheduler) scheduling_net_wrr(payloadSize uint32) int {
	return 0
}

func (c *SessionScheduler) scheduling_path() {

	if c.multipath {
		if c.numPath > 1 {
			c.usePath = c.numPath
		} else {
			// When numPath == 1 (There is no path to use multipath)
			return
		}
	} else {
		if c.usePath > 1 {
			c.usePath -= 1
		} else {
			// When usePath == 1 (Originally, it has used signle path)
			return
		}
	}

	if c.usePath > 1 {
		// When the additional path is added..
		// Reset remaining bytes of current path
		c.remainingBytes[c.currentPath] = c.weight[c.currentPath] * DATA_PACKET_PAYLOAD_SIZE
	}

	// Change current path to new path and Set the remainig bytes
	c.currentPath = c.usePath - 1
	remainBytesOfNewPath := c.weight[c.currentPath] * DATA_PACKET_PAYLOAD_SIZE

	c.remainingBytes = append(c.remainingBytes, remainBytesOfNewPath)

	util.Log("SessionScheduler.scheduling_path(): numPath=%d, usePath=%d, len remainingBytes=%d", c.numPath, c.usePath, len(c.remainingBytes))
}

// FIXME: To make a precise comparison, we need receiver information(throughput).
func (c *SessionScheduler) startMonitoring() {
	for {
		select {
		case <-c.monitor.tpChan:
			// If avg_throughput is less than.. (unit: Mbps)
			if c.monitor.compareThroughput() {
				if !c.multipath {
					c.multipath = true
					c.scheduling_path()
				}
			} else {
				if c.multipath {
					c.multipath = false
					c.scheduling_path()
				}
			}
		case <-c.monitor.bsChan:
			// If remaining_time is more than.. (unit: sec)
			if c.monitor.compareRemainingTime() {
				if !c.multipath {
					c.multipath = true
					c.scheduling_path()
				}
			} else {
				if c.multipath {
					c.multipath = false
					c.scheduling_path()
				}
			}
		}
	}
}
