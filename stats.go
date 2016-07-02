package main

import (
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// DurationStats represents statistics of durations
type DurationStats struct {
	maxDuration   time.Duration
	minDuration   time.Duration
	totalDuration time.Duration
}

// Stats represents the stats of the requests
type Stats struct {
	sync.Mutex
	nbOfRequests int
	durations    DurationStats
	statusStats  map[string]int
}

func newStats() *Stats {
	return &Stats{
		statusStats: map[string]int{},
	}
}

func (s *Stats) addError(err error) {
	s.Lock()
	defer s.Unlock()

	s.nbOfRequests++
	var errName string
	switch e := err.(type) {
	case *net.DNSError:
		errName = "DNS lookup error"
	case *net.DNSConfigError:
		errName = "DNS config error"
	case *net.AddrError:
		errName = "Addr Error"
	case *net.OpError:
		errName = "Op Error"
	case *url.Error:
		if e.Timeout() {
			errName = "URL Timeout"
		} else {
			errName = "Url Error"
		}
	case net.Error:
		errName = "Net Error"
	default:
		errName = err.Error()
	}
	s.statusStats[errName]++
}
func (s *Stats) addRequest(req *Request) {
	s.Lock()
	defer s.Unlock()
	s.nbOfRequests++
	s.durations.totalDuration += req.duration
	if s.durations.maxDuration < req.duration {
		s.durations.maxDuration = req.duration
	}
	if s.durations.minDuration == 0 || s.durations.minDuration > req.duration {
		s.durations.minDuration = req.duration
	}
	s.statusStats[http.StatusText(req.status)]++
}
