package lbproxy

import (
	"log"
	"sort"
	"sync"
	"time"
)

// unixTimeSupplier abstract retrieval of current time for testing harness
type unixTimeSupplier func() int64

type rlManager struct {
	sync.RWMutex
	tag                    string // for diagnostics
	config                 RateLimitManagerConfig
	currentOpenConnections int
	addedTimestamps        []int64
	currentTime            unixTimeSupplier
}

func CreateRateLimitManager(tag string, config RateLimitManagerConfig) *rlManager {
	rlm := &rlManager{
		tag:                    tag,
		config:                 config,
		currentOpenConnections: 0,
		addedTimestamps:        []int64{},
	}
	// Current time is normally wall time, but can be changed for testing
	rlm.currentTime = func() int64 {
		return time.Now().Unix()
	}
	return rlm
}

// overrideTimeSupplier is an internal method to supply a function to mock the passage of time for testing
func (m *rlManager) overrideTimeSupplier(supplier unixTimeSupplier) {
	m.currentTime = supplier
}

func (m *rlManager) AddConnection() bool {
	// Since most of the time we'll make writes, we'll just take one write lock
	m.Lock()
	defer m.Unlock()

	// If you have too many connections already open, deny
	if m.config.MaxOpenConnections >= 0 && m.currentOpenConnections >= m.config.MaxOpenConnections {
		log.Println("RLM", m.tag, "DENIED open:", m.currentOpenConnections, "max:", m.config.MaxOpenConnections)
		return false
	}

	// Only check added connection if we could possibly fail
	currentTs := m.currentTime()
	if m.config.MaxRateAmount >= 0 && len(m.addedTimestamps) >= m.config.MaxRateAmount {
		// +1 because if e.g. if we allow 1 event/sec, window will start at current time, because this timestamp has been already used
		windowStart := currentTs - m.config.MaxRatePeriodSeconds + 1

		// trim items outside of window
		m.addedTimestamps = trimTimestamps(m.addedTimestamps, windowStart)
		if len(m.addedTimestamps) >= m.config.MaxRateAmount {
			log.Println("RLM", m.tag, "DENIED @", currentTs, "ts:", m.addedTimestamps, "max:", m.config.MaxRateAmount)
			return false
		}
	}

	// If we got here, we're allowing the connection
	m.currentOpenConnections += 1

	// Only track added timestamps if connection rate-limiting is enabled, as the code above will limit inserts.
	// Without this check, we'll simply keep adding timestamps to the list when rate limiting is not enabled
	if m.config.MaxRateAmount >= 0 {
		m.addedTimestamps = append(m.addedTimestamps, currentTs)
	}
	log.Println("RLM+", m.tag, "open:", m.currentOpenConnections, "ts:", m.addedTimestamps)
	return true
}

func (m *rlManager) ReleaseConnection() {
	m.Lock()
	defer m.Unlock()
	if m.currentOpenConnections > 0 {
		m.currentOpenConnections -= 1
	}
	log.Println("RLM-", m.tag, "open:", m.currentOpenConnections, "ts:", m.addedTimestamps)
}

// trimTimestamps removed timestamps from ts that are older than windowStart
// elements in ts are added serially by current time, so it is an array guaranteed to be sorted
func trimTimestamps(ts []int64, windowStart int64) []int64 {
	// If last element is already too old, entire list can be cleared
	tsLen := len(ts)
	if tsLen == 0 || ts[tsLen-1] < windowStart {
		return []int64{}
	}

	// Use a standard binary search to find index of first item in window
	newStart := sort.Search(tsLen, func(i int) bool { return ts[i] >= windowStart })

	// Purge elements before windowStart
	return ts[newStart:]
}
