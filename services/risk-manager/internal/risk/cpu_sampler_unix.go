//go:build !windows

package risk

import (
	"runtime"
	"sync"
	"syscall"
	"time"
)

type cpuSampler struct {
	mu             sync.Mutex
	prevUserTime   int64
	prevSystemTime int64
	prevWall       time.Time
	numCPU         int
}

func newCPUSampler() *cpuSampler {
	return &cpuSampler{
		prevWall: time.Now(),
		numCPU:   runtime.GOMAXPROCS(0),
	}
}

func timevalToSeconds(tv syscall.Timeval) int64 {
	return int64(tv.Sec)*1e6 + int64(tv.Usec)
}

func (cs *cpuSampler) sample() float64 {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}

	now := time.Now()
	wallDelta := now.Sub(cs.prevWall).Seconds()
	if wallDelta <= 0 {
		cs.prevWall = now
		return 0
	}

	userNow := timevalToSeconds(ru.Utime)
	systemNow := timevalToSeconds(ru.Stime)
	userDelta := float64(userNow - cs.prevUserTime)
	systemDelta := float64(systemNow - cs.prevSystemTime)
	cpuSeconds := (userDelta + systemDelta) / 1e6

	cs.prevUserTime = userNow
	cs.prevSystemTime = systemNow
	cs.prevWall = now

	return (cpuSeconds / wallDelta) * 100.0 / float64(cs.numCPU)
}
