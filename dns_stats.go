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
	s.totalDuration += req.Duration()
	if s.maxDuration < req.Duration() {
		s.maxDuration = req.Duration()
	}
	if s.minDuration == 0 || s.minDuration > req.Duration() {
		s.minDuration = req.Duration()
	}
}

// Render renders the results
func (s *DNSStats) Render() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetHeader([]string{
		"Number of requests ",
		"Min duration",
		"Max duration",
		"Average duration",
		"Exec duration",
	})
	table.Append([]string{
		strconv.Itoa(s.nbOfRequests),
		s.minDuration.String(),
		s.maxDuration.String(),
		(s.totalDuration / time.Duration(s.nbOfRequests)).String(),
		s.execDuration.String(),
	})

	fmt.Printf("\nStats :\n")
	table.Render()

	statusTable := tablewriter.NewWriter(os.Stdout)
	statusTable.SetAlignment(tablewriter.ALIGN_CENTER)
	statusTable.SetHeader([]string{"Result", "Count"})
	for key, value := range s.statusStats {
		statusTable.Append([]string{key, strconv.Itoa(value)})
	}

	fmt.Printf("\nStatuses :\n")
	statusTable.Render()
}

// SetDuration will set the total duration of the simulation
func (s *DNSStats) SetDuration(t time.Duration) {
	s.execDuration = t
}
