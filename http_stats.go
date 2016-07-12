package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
)

// HTTPStats represents the stats of the requests
type HTTPStats struct {
	DurationStats
	sync.Mutex
	nbOfRequests int
	statusStats  map[string]int
	totalSize    int64
}

// newHTTPStats will return an empty Stats object
func newHTTPStats() Stats {
	return &HTTPStats{
		DurationStats: DurationStats{},
		statusStats:   map[string]int{},
	}
}

// AddRequest will add a request to the stats
func (s *HTTPStats) AddRequest(req Request) {
	s.Lock()
	defer s.Unlock()
	s.nbOfRequests++
	s.addDuration(req)

	if req.IsError() {
		s.statusStats[req.Error()]++
		return
	}
	s.statusStats[req.Status()]++
	s.totalSize += req.Size()
}

// addDuration will add the duration of a requests to the stats
func (s *HTTPStats) addDuration(req Request) {
	s.totalDuration += req.Duration()
	if s.maxDuration < req.Duration() {
		s.maxDuration = req.Duration()
	}
	if s.minDuration == 0 || s.minDuration > req.Duration() {
		s.minDuration = req.Duration()
	}
}

// Render renders the results
func (s *HTTPStats) Render() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetHeader([]string{
		"Number of requests ",
		"Min duration",
		"Max duration",
		"Average duration",
		"Exec duration",
		"Avg speed",
		"Total size",
	})
	table.Append([]string{
		strconv.Itoa(s.nbOfRequests),
		s.minDuration.String(),
		s.maxDuration.String(),
		(s.totalDuration / time.Duration(s.nbOfRequests)).String(),
		s.execDuration.String(),
		fmt.Sprintf("%s/s", humanize.Bytes(uint64(float64(time.Duration(s.totalSize))/float64(s.execDuration)*math.Pow10(9)))),
		fmt.Sprintf("%s", humanize.Bytes(uint64(s.totalSize))),
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
func (s *HTTPStats) SetDuration(t time.Duration) {
	s.execDuration = t
}
