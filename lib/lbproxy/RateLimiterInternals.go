package lbproxy

import (
	"sync"
	"time"
)

type rlManager struct {
	sync.RWMutex
	config                 RateLimitManagerConfig
	currentOpenConnections int
	addedTimestamps        []int64
}

func CreateRateLimitManager(config RateLimitManagerConfig) RateLimitManager {
	return &rlManager{
		config:                 config,
		currentOpenConnections: 0,
		addedTimestamps:        []int64{},
	}
}

func (m *rlManager) AddConnection() bool {
	// Since most of the time we'll make writes, we'll just take one write lock
	m.Lock()
	defer m.Unlock()

	// If you have too many connections already open, deny
	if m.currentOpenConnections >= m.config.MaxOpenConnections {
		return false
	}

	// Only check added connection if we could possibly fail
	currentTs := time.Now().Unix()
	if len(m.addedTimestamps) >= m.config.MaxRateAmount {
		windowStart := currentTs - m.config.MaxRatePeriodSeconds
		// TODO: off by one? First retry pretty quickly always fails?
		// trim items outside of window
		m.addedTimestamps = trimTimestamps(m.addedTimestamps, windowStart)
		if len(m.addedTimestamps) >= m.config.MaxRateAmount {
			return false
		}
	}

	// If we got here, we're allowing the connection
	m.currentOpenConnections += 1
	m.addedTimestamps = append(m.addedTimestamps, currentTs)
	return true
}

func (m *rlManager) ReleaseConnection() {
	m.Lock()
	if m.currentOpenConnections > 0 {
		m.currentOpenConnections -= 1
	}
	m.Unlock()
}

func trimTimestamps(ts []int64, windowStart int64) []int64 {
	newStart := 0
	for i, t := range ts {
		newStart = i
		if t >= windowStart {
			break
		}
	}
	if newStart >= len(ts) || ts[newStart] < windowStart {
		return []int64{}
	}
	return ts[newStart:]
}
