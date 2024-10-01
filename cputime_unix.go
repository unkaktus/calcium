package calcium

import (
	"syscall"
	"time"
)

type CPUTime struct {
	User   time.Duration
	System time.Duration
}

func GetCPUTime() (*CPUTime, error) {
	rusage := &syscall.Rusage{}
	if err := syscall.Getrusage(syscall.RUSAGE_CHILDREN, rusage); err != nil {
		return nil, err
	}
	cpuTime := &CPUTime{
		System: time.Duration(rusage.Stime.Nano()),
		User:   time.Duration(rusage.Utime.Nano()),
	}

	return cpuTime, nil
}
