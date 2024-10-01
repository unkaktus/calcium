package cputime

import (
	"syscall"
	"time"
)

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
