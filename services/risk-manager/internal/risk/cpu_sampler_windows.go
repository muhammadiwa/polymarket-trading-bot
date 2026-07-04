//go:build windows

package risk

import (
	"runtime"
	"syscall"
	"time"
)

type cpuSampler struct {
	prevUserTime   int64
	prevKernelTime int64
	prevWall       time.Time
	numCPU         int
}

func newCPUSampler() *cpuSampler {
	return &cpuSampler{
		prevWall: time.Now(),
		numCPU:   runtime.GOMAXPROCS(0),
	}
}

func filetimeTo100ns(ft syscall.Filetime) int64 {
	return int64(ft.HighDateTime)<<32 | int64(ft.LowDateTime)
}

func (cs *cpuSampler) sample() float64 {
	handle, _ := syscall.GetCurrentProcess()
	var creation, exit, kernel, user syscall.Filetime
	if err := syscall.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err != nil {
		return 0
	}

	now := time.Now()
	wallDelta := now.Sub(cs.prevWall).Seconds()
	if wallDelta <= 0 {
		cs.prevWall = now
		return 0
	}

	userNow := filetimeTo100ns(user)
	kernelNow := filetimeTo100ns(kernel)
	userDelta := float64(userNow - cs.prevUserTime)
	kernelDelta := float64(kernelNow - cs.prevKernelTime)
	cpuSeconds := (userDelta + kernelDelta) / 1e7

	cs.prevUserTime = userNow
	cs.prevKernelTime = kernelNow
	cs.prevWall = now

	return (cpuSeconds / wallDelta) * 100.0 / float64(cs.numCPU)
}
