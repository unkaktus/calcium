package cputime

import (
	"fmt"
	"syscall"
	"time"
)

func GetCPUTime() (*CPUTime, error) {
	tms := syscall.Tms{}
	_, err := syscall.Times(&tms)
	if err != nil {
		return nil, fmt.Errorf("syscall Times: %w", err)
	}
	cpuTime := &CPUTime{
		User:   time.Duration(float64(tms.Utime+tms.Cutime)*10) * time.Millisecond,
		System: time.Duration(float64(tms.Stime+tms.Cstime)*10) * time.Millisecond,
	}
	return cpuTime, nil
}
