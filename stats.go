package main

import "time"

// Stats represents a statistics interface
type Stats interface {
	AddRequest(Request)
	Render()
	SetDuration(time.Duration)
}

// DurationStats represents statistics of durations
type DurationStats struct {
	maxDuration   time.Duration
	minDuration   time.Duration
	totalDuration time.Duration
	execDuration  time.Duration
}

func newStats(trafficType string) (Stats, error) {
	stats, ok := statsMap[trafficType]
	if !ok {
		return nil, ErrInvalidTrafficType
	}
	return stats(), nil
}
