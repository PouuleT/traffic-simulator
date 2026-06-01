package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	maxStatusDisplay = 5 // Maximum status codes to display before truncating
)

// renderLiveView renders the live traffic generation view
func (m *tuiModel) renderLiveView() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Stats panel
	b.WriteString(m.renderStatsPanel())
	b.WriteString("\n")

	// Log separator
	b.WriteString(logSeparatorStyle.Render("─ Request Log " + strings.Repeat("─", max(0, m.width-16))))
	b.WriteString("\n")

	// Request log
	b.WriteString(m.renderRequestLog())

	return b.String()
}

// renderHeader renders the header with config and progress
func (m *tuiModel) renderHeader() string {
	elapsed := time.Since(m.startTime).Round(100 * time.Millisecond)

	title := fmt.Sprintf("Traffic Simulator ── %s ── %d clients × %d requests",
		strings.ToUpper(m.trafficType),
		m.cfg.NbOfClients,
		m.cfg.NbOfRequests,
	)

	// Progress bar
	percent := 0.0
	if m.totalReqs > 0 {
		percent = float64(m.completed) / float64(m.totalReqs)
	}

	progressBar := m.progress.ViewAs(percent)
	progressText := fmt.Sprintf("%d/%d (%.0f%%)   ⏱ %s",
		m.completed, m.totalReqs, percent*100, elapsed)

	return titleStyle.Render(title) + "\n\n" +
		"  " + progressBar + "  " + progressText
}

// renderStatsPanel renders the stats boxes side by side
func (m *tuiModel) renderStatsPanel() string {
	statsBox := m.renderStatsBox()
	statusBox := m.renderStatusBox()

	// Side by side layout
	return lipgloss.JoinHorizontal(lipgloss.Top, statsBox, statusBox)
}

// renderStatsBox renders the live stats box
func (m *tuiModel) renderStatsBox() string {
	snapshot := m.getStatsSnapshot()

	elapsed := time.Since(m.startTime).Seconds()
	reqPerSec := 0.0
	if elapsed > 0 && m.completed > 0 {
		reqPerSec = float64(m.completed) / elapsed
	}

	content := fmt.Sprintf(
		"Requests/sec:  %.1f\n"+
			"Avg latency:   %s\n"+
			"Min latency:   %s\n"+
			"Max latency:   %s\n"+
			"Total size:    %s\n"+
			"Throughput:    %s/s",
		reqPerSec,
		snapshot.avgDuration,
		snapshot.minDuration,
		snapshot.maxDuration,
		snapshot.totalSize,
		snapshot.throughput,
	)

	return boxStyle.Render("Live Stats\n\n" + content)
}

// renderStatusBox renders the status code breakdown
func (m *tuiModel) renderStatusBox() string {
	snapshot := m.getStatsSnapshot()

	var lines []string
	displayCount := len(snapshot.statusCodes)

	// In live view, limit to maxStatusDisplay; in final view, show all
	if !m.done && displayCount > maxStatusDisplay {
		displayCount = maxStatusDisplay
	}

	for i := 0; i < displayCount; i++ {
		status := snapshot.statusCodes[i]
		// Create horizontal bar
		barLength := status.count * 10 / max(1, m.completed)
		if barLength < 0 {
			barLength = 0
		}
		if barLength > 10 {
			barLength = 10
		}
		bar := strings.Repeat("█", barLength)

		style := getStatusStyle(status.name, status.isError)
		line := fmt.Sprintf("%-20s %s %3d", status.name, style.Render(bar), status.count)
		lines = append(lines, line)
	}

	// Add truncation notice if needed (only in live view)
	if !m.done && len(snapshot.statusCodes) > maxStatusDisplay {
		remaining := len(snapshot.statusCodes) - maxStatusDisplay
		lines = append(lines, grayStyle.Render(fmt.Sprintf("...and %d more", remaining)))
	}

	content := strings.Join(lines, "\n")
	if content == "" {
		content = "No requests completed yet"
	}

	return boxStyle.Render("Status Codes\n\n" + content)
}

// renderRequestLog renders the scrolling request log
func (m *tuiModel) renderRequestLog() string {
	if len(m.requestLog) == 0 {
		return grayStyle.Render("  Waiting for requests...\n")
	}

	var lines []string
	for _, entry := range m.requestLog {
		lines = append(lines, m.formatLogEntry(entry))
	}

	// Show last N lines that fit
	maxLines := max(1, m.height-18) // Reserve space for header and stats
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	return strings.Join(lines, "\n")
}

// formatLogEntry formats a single request log entry
func (m *tuiModel) formatLogEntry(entry requestLogEntry) string {
	statusStyle := getStatusStyle(entry.status, entry.isError)
	latencyMs := entry.duration.Milliseconds()
	latencyStyle := getLatencyStyle(latencyMs)

	// Truncate URL if needed
	url := entry.url
	if len(url) > 50 {
		url = url[:47] + "..."
	}

	return fmt.Sprintf("  worker#%02d  %02d/%02d  %s  %s  %-50s  %s",
		entry.workerID,
		entry.reqNumber,
		entry.totalReqs,
		statusStyle.Render(fmt.Sprintf("%-3s", entry.status)),
		latencyStyle.Render(fmt.Sprintf("%6s", entry.duration.Round(time.Millisecond))),
		url,
		entry.sizeOrError,
	)
}

// renderFinalSummary renders the final summary screen
func (m *tuiModel) renderFinalSummary() string {
	var b strings.Builder

	elapsed := m.completionTime.Round(100 * time.Millisecond)

	b.WriteString(titleStyle.Render("Traffic Simulator ── Complete"))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "  ✓ %d requests completed in %s (%d clients)\n\n",
		m.completed, elapsed, m.cfg.NbOfClients)

	// Summary boxes
	summaryBox := m.renderSummaryBox()
	statusBox := m.renderStatusBox()

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, summaryBox, statusBox))
	b.WriteString("\n")

	// Timeline (HTTP only)
	if m.trafficType == "http" {
		b.WriteString(m.renderTimelineBox())
		b.WriteString("\n")
	}

	// Quit prompt
	quitPrompt := grayStyle.Render("\n  Press 'q' or Enter to quit...")
	b.WriteString(quitPrompt)
	b.WriteString("\n")

	return b.String()
}

// renderSummaryBox renders the summary stats box
func (m *tuiModel) renderSummaryBox() string {
	snapshot := m.getStatsSnapshot()

	content := fmt.Sprintf(
		"Total requests:  %d\n"+
			"Success rate:    %s\n"+
			"Avg latency:     %s\n"+
			"Min latency:     %s\n"+
			"Max latency:     %s\n"+
			"Avg speed:       %s/s\n"+
			"Total size:      %s",
		m.completed,
		snapshot.successRate,
		snapshot.avgDuration,
		snapshot.minDuration,
		snapshot.maxDuration,
		snapshot.throughput,
		snapshot.totalSize,
	)

	return boxStyle.Render("Summary\n\n" + content)
}

// renderTimelineBox renders the HTTP response timeline
func (m *tuiModel) renderTimelineBox() string {
	snapshot := m.getStatsSnapshot()

	if len(snapshot.timeline) == 0 {
		return ""
	}

	var lines []string
	for _, step := range snapshot.timeline {
		// Create horizontal bar
		barLength := int(step.durationMs / max64(1, 10)) // Scale to fit
		if barLength < 0 {
			barLength = 0
		}
		if barLength > 20 {
			barLength = 20
		}
		bar := strings.Repeat("█", barLength)

		line := fmt.Sprintf("%-20s %s %s", step.name, successStyle.Render(bar), step.duration)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	return boxStyle.Render("Response Timeline (avg)\n\n" + content)
}

// statsSnapshot holds a snapshot of stats for rendering
type statsSnapshot struct {
	avgDuration string
	minDuration string
	maxDuration string
	totalSize   string
	throughput  string
	successRate string
	statusCodes []statusCount
	timeline    []timelineStep
}

type statusCount struct {
	name    string
	count   int
	isError bool
}

type timelineStep struct {
	name       string
	duration   string
	durationMs int64
}

// getStatsSnapshot gets a thread-safe snapshot of current stats
func (m *tuiModel) getStatsSnapshot() statsSnapshot {
	data := m.stats.Snapshot()

	// Use completion time if done, otherwise current elapsed time
	var elapsed float64
	if m.done {
		elapsed = m.completionTime.Seconds()
	} else {
		elapsed = time.Since(m.startTime).Seconds()
	}

	// Calculate throughput
	throughput := "0 B"
	if elapsed > 0 && data.TotalSize > 0 {
		bytesPerSec := uint64(float64(data.TotalSize) / elapsed)
		throughput = humanizeBytes(bytesPerSec)
	}

	// Calculate success rate
	successRate := "0%"
	if data.NbOfRequests > 0 {
		// For HTTP, use SuccessRequests. For DNS, calculate from statusStats
		successCount := data.SuccessRequests
		if successCount == 0 {
			// Count non-error statuses
			for status, count := range data.StatusStats {
				if status == "OK" || status == "OK " {
					successCount += count
				}
			}
		}
		rate := float64(successCount) * 100.0 / float64(data.NbOfRequests)
		successRate = fmt.Sprintf("%.0f%%", rate)
	}

	// Build status codes list
	var statusCodes []statusCount
	for name, count := range data.StatusStats {
		isError := strings.Contains(name, "error") ||
			strings.Contains(name, "Error") ||
			strings.Contains(name, "timeout") ||
			strings.Contains(name, "Timeout")
		statusCodes = append(statusCodes, statusCount{
			name:    name,
			count:   count,
			isError: isError,
		})
	}

	// Sort by count descending
	sortStatusCodes(statusCodes)

	// Build timeline
	var timeline []timelineStep
	if data.ResponseTimeline != nil && data.SuccessRequests > 0 {
		timeline = []timelineStep{
			{
				name:       "DNS Lookup",
				duration:   getAvgDuration(data.ResponseTimeline.DNSLookup, data.SuccessRequests),
				durationMs: data.ResponseTimeline.DNSLookup.Milliseconds() / int64(max(1, data.SuccessRequests)),
			},
			{
				name:       "TCP Connection",
				duration:   getAvgDuration(data.ResponseTimeline.TCPConnection, data.SuccessRequests),
				durationMs: data.ResponseTimeline.TCPConnection.Milliseconds() / int64(max(1, data.SuccessRequests)),
			},
			{
				name:       "TLS Handshake",
				duration:   getAvgDuration(data.ResponseTimeline.EstablishingConnection, data.SuccessRequests),
				durationMs: data.ResponseTimeline.EstablishingConnection.Milliseconds() / int64(max(1, data.SuccessRequests)),
			},
			{
				name:       "Server Processing",
				duration:   getAvgDuration(data.ResponseTimeline.ServerProcessing, data.SuccessRequests),
				durationMs: data.ResponseTimeline.ServerProcessing.Milliseconds() / int64(max(1, data.SuccessRequests)),
			},
			{
				name:       "Content Transfer",
				duration:   getAvgDuration(data.ResponseTimeline.ContentTransfer, data.SuccessRequests),
				durationMs: data.ResponseTimeline.ContentTransfer.Milliseconds() / int64(max(1, data.SuccessRequests)),
			},
		}
	}

	return statsSnapshot{
		avgDuration: getAvgDuration(data.TotalDuration, data.NbOfRequests),
		minDuration: data.MinDuration.String(),
		maxDuration: data.MaxDuration.String(),
		totalSize:   humanizeBytes(uint64(data.TotalSize)),
		throughput:  throughput,
		successRate: successRate,
		statusCodes: statusCodes,
		timeline:    timeline,
	}
}

// sortStatusCodes sorts status codes by count descending, then alphabetically
func sortStatusCodes(codes []statusCount) {
	// Simple bubble sort for small lists
	for i := 0; i < len(codes); i++ {
		for j := i + 1; j < len(codes); j++ {
			// Sort by count descending, then alphabetically
			if codes[i].count < codes[j].count ||
				(codes[i].count == codes[j].count && codes[i].name > codes[j].name) {
				codes[i], codes[j] = codes[j], codes[i]
			}
		}
	}
}

// humanizeBytes formats bytes in human-readable format
func humanizeBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
