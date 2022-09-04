package lbproxy

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

// How many times to repeat parallel tests, to ensure results are stable
const TestRepeatCount = 5

func Test_RateLimitManagerConnectRates(t *testing.T) {
	// NOTE: these tests cannot give consistent results in parallel, as the goroutines will increment time
	// while adding connections in an arbitrary order, adding a degree of variability

	// Basic test: allowing 1/sec
	t.Run("basicRate", func(t *testing.T) {
		rlm, now := newTestRLM(-1, 1, 1)
		allowedCount := 0

		iterations := 30
		rate := 3

		for i := 1; i <= iterations; i++ { // Start at 1 so %0 doesn't hit right away
			if rlm.AddConnection() {
				allowedCount++
			}
			// Increment every third item; i.e. rate 3 per sec
			if i%rate == 0 {
				now.Add(1)
			}
		}

		// With 30 iterations at 3 per sec, we can add 10 items
		goal := iterations / rate
		if allowedCount != goal {
			t.Errorf("basicRate total allowed = %v, want %v", allowedCount, goal)
		}
	})

	// High rate per second
	t.Run("highRate", func(t *testing.T) {
		maxRate := 5
		rlm, now := newTestRLM(-1, maxRate, 1)
		allowedCount := 0
		iterations := 100
		actualRate := 10

		for i := 1; i <= iterations; i++ { // Start at 1 so %0 doesn't hit right away
			if rlm.AddConnection() {
				allowedCount++
			}

			if i%actualRate == 0 {
				now.Add(1)
			}
		}

		// We get the fraction of tries that matches the allowed rate (works with multiples only since integer division here)
		goal := iterations / (actualRate / maxRate)
		if allowedCount != goal {
			t.Errorf("highRate total allowed = %v, want %v", allowedCount, goal)
		}
	})

	// Longer period
	t.Run("longPeriod", func(t *testing.T) {
		ratePeriod := 20
		rlm, now := newTestRLM(-1, 1, ratePeriod)
		allowedCount := 0

		iterations := 60

		for i := 1; i <= iterations; i++ { // Start at 1 so %0 doesn't hit right away
			if rlm.AddConnection() {
				allowedCount++
			}
			now.Add(1)
		}

		// Adding every second, rate period is the limit (works with multiples only since integer division here)
		goal := iterations / ratePeriod
		if allowedCount != goal {
			t.Errorf("longPeriod total allowed = %v, want %v", allowedCount, goal)
		}
	})
}

func Test_RateLimitManagerMaxOpen(t *testing.T) {
	maxOpen := 3

	// Add too many connections (in parallel), and ensure only up to maxOpen are added
	t.Run("maxOpenRespected", func(t *testing.T) {
		for repeat := 0; repeat < TestRepeatCount; repeat++ {
			rlm, _ := newTestRLM(maxOpen, -1, 0)
			allowedCount := atomic.Int32{}
			allowedCount.Store(0)

			wg := sync.WaitGroup{}
			iterations := maxOpen * 3
			wg.Add(iterations)
			for i := 0; i < iterations; i++ {
				go func() {
					if rlm.AddConnection() {
						allowedCount.Add(1)
					}
					wg.Done()
				}()
			}
			wg.Wait()
			if allowedCount.Load() != int32(maxOpen) {
				t.Errorf("maxOpenRespected total allowed = %v, want %v", allowedCount.Load(), maxOpen)
			}
		}
	})

	// Add then release connections (in parallel); because of serialized access, every single one should be allowed
	t.Run("maxOpenRespectedWithRelease", func(t *testing.T) {
		for repeat := 0; repeat < TestRepeatCount; repeat++ {
			rlm, _ := newTestRLM(maxOpen, -1, 0)
			allowedCount := atomic.Int32{}
			allowedCount.Store(0)

			wg := sync.WaitGroup{}
			iterations := maxOpen * 3
			wg.Add(iterations)
			for i := 0; i < iterations; i++ {
				go func(i int) {
					// If allowed, release
					if rlm.AddConnection() {
						allowedCount.Add(1)
						rlm.ReleaseConnection()
					}
					wg.Add(-1)
				}(i)
			}
			wg.Wait()
			if allowedCount.Load() != int32(iterations) {
				t.Errorf("maxOpenRespectedWithRelease total allowed = %v, want %v", allowedCount.Load(), iterations)
			}
		}
	})
}

func newTestRLM(maxOpen int, maxRate int, ratePeriodSec int) (RateLimitManager, *atomic.Int64) {
	rlm := CreateRateLimitManager("ut", RateLimitManagerConfig{
		MaxOpenConnections:   maxOpen,
		MaxRateAmount:        maxRate,
		MaxRatePeriodSeconds: int64(ratePeriodSec),
	})

	currentTime := atomic.Int64{}
	currentTime.Store(1)
	rlm.overrideTimeSupplier(func() int64 {
		return currentTime.Load()
	})
	return rlm, &currentTime
}

func Test_trimTimestamps(t *testing.T) {
	type args struct {
		ts          []int64
		windowStart int64
	}
	tests := []struct {
		name string
		args args
		want []int64
	}{
		{
			name: "empty",
			args: args{
				ts:          []int64{},
				windowStart: 100,
			},
			want: []int64{},
		},
		{
			name: "single in",
			args: args{
				ts:          []int64{101},
				windowStart: 100,
			},
			want: []int64{101},
		},
		{
			name: "single out",
			args: args{
				ts:          []int64{99},
				windowStart: 100,
			},
			want: []int64{},
		},
		{
			name: "slice",
			args: args{
				ts:          []int64{99, 100, 101},
				windowStart: 100,
			},
			want: []int64{100, 101},
		},
		{
			name: "noop",
			args: args{
				ts:          []int64{99, 100, 101},
				windowStart: 90,
			},
			want: []int64{99, 100, 101},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimTimestamps(tt.args.ts, tt.args.windowStart); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("trimTimestamps() = %v, want %v", got, tt.want)
			}
		})
	}
}
