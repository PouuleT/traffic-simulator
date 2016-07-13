package main

import (
	"time"

	"github.com/fatih/color"
)

const (
	// Success is a successful request
	Success criticityLevel = iota
	// Warning is a successful request but with a bad status
	Warning
	// Critical is an unsuccessful request
	Critical
)

// criticityLevel represents the criticity level of a request
type criticityLevel int

// Colors
var red = color.New(color.FgRed).SprintfFunc()
var green = color.New(color.FgGreen).SprintfFunc()
var yellow = color.New(color.FgYellow).SprintfFunc()

var criticityColor = map[criticityLevel]func(string, ...interface{}) string{
	Success:  green,
	Warning:  yellow,
	Critical: red,
}

// Request represents a Request interface
type Request interface {
	String() string
	Error() string
	IsError() bool
	Status() string
	Size() int64
	Duration() time.Duration
}
