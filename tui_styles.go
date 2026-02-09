package main

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	colorGreen  = lipgloss.Color("#00ff00")
	colorYellow = lipgloss.Color("#ffff00")
	colorRed    = lipgloss.Color("#ff0000")
	colorCyan   = lipgloss.Color("#00ffff")
	colorGray   = lipgloss.Color("#888888")

	grayStyle = lipgloss.NewStyle().Foreground(colorGray)

	// Header styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan).
			MarginLeft(1)

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorCyan).
			Padding(0, 1).
			MarginRight(1)

	// Status styles
	successStyle = lipgloss.NewStyle().Foreground(colorGreen)
	warningStyle = lipgloss.NewStyle().Foreground(colorYellow)
	errorStyle   = lipgloss.NewStyle().Foreground(colorRed)

	// Log separator
	logSeparatorStyle = lipgloss.NewStyle().
				Foreground(colorCyan).
				MarginTop(1).
				MarginBottom(1)
)

// getStatusStyle returns the appropriate style for a status code
func getStatusStyle(status string, isError bool) lipgloss.Style {
	if isError {
		return errorStyle
	}

	// HTTP status codes
	if len(status) >= 1 {
		switch status[0] {
		case '2': // 2xx success
			return successStyle
		case '3', '4': // 3xx redirect, 4xx client error
			return warningStyle
		case '5': // 5xx server error
			return errorStyle
		}
	}

	return successStyle
}

// getLatencyStyle returns the appropriate style for latency
func getLatencyStyle(ms int64) lipgloss.Style {
	if ms < 200 {
		return successStyle
	} else if ms < 1000 {
		return warningStyle
	}
	return errorStyle
}
