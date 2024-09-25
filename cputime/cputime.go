package cputime

import "time"

type CPUTime struct {
	User   time.Duration
	System time.Duration
}
