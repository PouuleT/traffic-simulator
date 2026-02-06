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

// updateDuration updates the duration statistics with a new request duration
func (d *DurationStats) updateDuration(dur time.Duration) {
	d.totalDuration += dur
	if d.maxDuration < dur {
		d.maxDuration = dur
	}
	if d.minDuration == 0 || d.minDuration > dur {
		d.minDuration = dur
	}
}

func newStats(trafficType string) (Stats, error) {
	stats, ok := statsMap[trafficType]
	if !ok {
		return nil, ErrInvalidTrafficType
	}
	return stats(), nil
}
