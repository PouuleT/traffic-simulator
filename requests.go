package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/fatih/color"
)

// Colors
var red = color.New(color.FgRed).SprintfFunc()
var green = color.New(color.FgGreen).SprintfFunc()
var yellow = color.New(color.FgYellow).SprintfFunc()

// Request represents a requsest reponse, with the return code and the duration
type Request struct {
	status      string
	statusShort string
	url         string
	criticity   criticityLevel
	duration    time.Duration
	err         error
}

type criticityLevel int

const (
	Success  criticityLevel = iota
	Warning  criticityLevel = iota
	Critical criticityLevel = iota
)

var criticityColor = map[criticityLevel]func(string, ...interface{}) string{
	Success:  green,
	Warning:  yellow,
	Critical: red,
}

// findRandomURL will return a random URL
func findRandomURL() string {
	return URLs[rand.Intn(len(URLs))]
}

// getURL will get a given URL and return a *Request
func getURL(url string) *Request {
	client := http.DefaultClient
	client.Timeout = time.Duration(timeout) * time.Second
	url = "http://" + url

	var dur time.Duration
	t := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		dur = time.Since(t)
		return &Request{
			url:       url,
			duration:  dur,
			err:       err,
			criticity: Critical,
		}
	}
	defer resp.Body.Close()

	// Read the full body
	io.Copy(ioutil.Discard, resp.Body)
	// Record the duration of the request
	dur = time.Since(t)

	// If some http statuses has no StatusText, return a simple string with the
	// http status
	statusText := http.StatusText(resp.StatusCode)
	if statusText == "" {
		statusText = fmt.Sprintf("%d", resp.StatusCode)
	}

	var reqCriticity criticityLevel
	if resp.StatusCode == http.StatusOK {
		reqCriticity = Success
	} else {
		reqCriticity = Warning
	}

	return &Request{
		url:         url,
		duration:    dur,
		status:      statusText,
		statusShort: strconv.Itoa(resp.StatusCode),
		criticity:   reqCriticity,
	}
}

// lookupURL will make a DNS request on a given URL and return a *Request
func lookupURL(url string) *Request {
	var dur time.Duration
	t := time.Now()
	// Make the DNS request
	_, err := net.LookupHost(url)
	if err != nil {
		dur = time.Since(t)
		return &Request{
			duration:  dur,
			err:       err,
			criticity: Critical,
		}
	}

	// Record the duration of the request
	dur = time.Since(t)

	return &Request{
		duration:  dur,
		status:    "OK ",
		criticity: Success,
		url:       url,
	}
}

// Log will return the string to display on a log
func (r *Request) Log() string {
	// If there is a short status, print it instead of a long one
	statusToShow := r.statusShort
	if statusToShow == "" {
		statusToShow = r.status
	}
	return fmt.Sprintf("| %s | %12s | Get %s", criticityColor[r.criticity](statusToShow), r.duration, r.url)
}
