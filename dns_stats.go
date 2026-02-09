package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/olekukonko/tablewriter"
)

// DNSStats represents the stats of the requests
type DNSStats struct {
	DurationStats
	sync.Mutex
	nbOfRequests int
	statusStats  map[string]int
}

// newDNSStats will return an empty Stats object
func newDNSStats() Stats {
	return &DNSStats{
		DurationStats: DurationStats{},
		statusStats:   map[string]int{},
	}
}

// AddRequest will add a request to the stats
func (s *DNSStats) AddRequest(req Request) {
	s.Lock()
	defer s.Unlock()
	s.nbOfRequests++
	s.addDuration(req)

	if req.IsError() {
		s.statusStats[req.Error()]++
		return
	}
	s.statusStats[req.Status()]++
}

// addDuration will add the duration of a requests to the stats
func (s *DNSStats) addDuration(req Request) {
	s.updateDuration(req.Duration())
}

// Render renders the results
func (s *DNSStats) Render() {
	table := tablewriter.NewTable(os.Stdout)
	table.Header([]string{
		"Number of requests ",
		"Min duration",
		"Max duration",
		"Average duration",
		"Exec duration",
	})
	_ = table.Append([]string{
		strconv.Itoa(s.nbOfRequests),
		s.minDuration.String(),
		s.maxDuration.String(),
		getAvgDuration(s.totalDuration, s.nbOfRequests),
		s.execDuration.String(),
	})

	fmt.Printf("\nStats :\n")
	_ = table.Render()

	statusTable := tablewriter.NewTable(os.Stdout)
	statusTable.Header([]string{"Result", "Count"})
	for key, value := range s.statusStats {
		_ = statusTable.Append([]string{key, strconv.Itoa(value)})
	}

	fmt.Printf("\nStatuses :\n")
	_ = statusTable.Render()
}

// SetDuration will set the total duration of the simulation
func (s *DNSStats) SetDuration(t time.Duration) {
	s.execDuration = t
}

// Snapshot returns a thread-safe snapshot of current stats
func (s *DNSStats) Snapshot() StatsData {
	s.Lock()
	defer s.Unlock()

	// Copy the map
	statusCopy := make(map[string]int, len(s.statusStats))
	for k, v := range s.statusStats {
		statusCopy[k] = v
	}

	return StatsData{
		NbOfRequests:     s.nbOfRequests,
		SuccessRequests:  0, // DNS doesn't track success separately
		StatusStats:      statusCopy,
		MinDuration:      s.minDuration,
		MaxDuration:      s.maxDuration,
		TotalDuration:    s.totalDuration,
		ExecDuration:     s.execDuration,
		TotalSize:        0,
		ResponseTimeline: nil,
	}
}
