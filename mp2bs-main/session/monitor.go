package session

import (
	"mp2bs/config"
	"time"
)

type SessionMonitor struct {
	remainingTime   float64
	avgThroughput   float64
	tpChan          chan bool
	bsChan          chan bool
	totalBytes      uint32
	throughput      float64
	startTime       time.Time
	prevElapsedTime time.Duration
}

func CreateSessionMonitor() *SessionMonitor {
	// Create SessionMonitor
	sm := SessionMonitor{
		remainingTime:   0,
		avgThroughput:   0,
		tpChan:          make(chan bool),
		bsChan:          make(chan bool),
		totalBytes:      0,
		throughput:      0,
		startTime:       time.Now(),
		prevElapsedTime: time.Since(time.Now()),
	}
	return &sm
}

func (m *SessionMonitor) compareThroughput() bool {
	if (m.avgThroughput / (1000 * 1000)) < config.Conf.MULTIPATH_THRESHOLD_THROUGHPUT {
		// multipath
		return true
	} else {
		// single path
		return false
	}
}

func (m *SessionMonitor) compareRemainingTime() bool {
	if m.remainingTime > config.Conf.MULTIPATH_THRESHOLD_SEND_COMPLETE_TIME {
		// multipath
		return true
	} else {
		// singel path
		return false
	}
}

func (m *SessionMonitor) computeThroughput(sentBytes uint32, currentTime time.Time) {
	elapsedTime := currentTime.Sub(m.startTime)

	if float32(elapsedTime.Seconds()) > config.Conf.THROUGHPUT_PERIOD {
		m.throughput = (float64(m.totalBytes) * 8.0) / float64(m.prevElapsedTime.Seconds()) // bps

		m.avgThroughput = float64(config.Conf.THROUGHPUT_WEIGHT)*m.avgThroughput + float64((1-config.Conf.THROUGHPUT_WEIGHT))*m.throughput

		// util.Log("monitor.computeThroughput(): avgThroughput=%f bps, throughput=%f, preveElapsedTime=%f\n", m.avgThroughput, m.throughput, m.prevElapsedTime.Seconds())
		// fmt.Printf("monitor.computeThroughput(): avgThroughput=%f bps, throughput=%f, preveElapsedTime=%f\n", m.avgThroughput, m.throughput, m.prevElapsedTime.Seconds())

		m.tpChan <- true

		m.totalBytes = sentBytes
		m.startTime = currentTime
	} else {
		m.totalBytes += sentBytes
		m.prevElapsedTime = elapsedTime
	}

}

func (m *SessionMonitor) computeRemainingTime(remainingBytes uint32) {
	m.remainingTime = float64(remainingBytes) / m.avgThroughput

	// util.Log("monitor.computeremainingTime(): avgThroughput=%f bps, remainingTime=%f, remainingBytes=%d", m.avgThroughput, m.remainingTime, remainingBytes)
	// fmt.Printf("monitor.computeremainingTime(): avgThroughput=%f bps, remainingTime=%f, remainingBytes=%d\n", m.avgThroughput, m.remainingTime, remainingBytes)

	m.bsChan <- true
}
