package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
)

// HTTPRequest represents a request response, with the return code and the duration
type HTTPRequest struct {
	status      string
	statusShort string
	url         string
	criticity   criticityLevel
	duration    time.Duration
	err         error
	size        int64
}

// String will return the string representing the request
func (r *HTTPRequest) String() string {
	if r.IsError() {
		return fmt.Sprintf("| %s | %12s | Get %s : %s", red("ERR"), r.duration, r.url, r.Error())
	}
	return fmt.Sprintf("| %s | %12s | Get %s ( %s )", criticityColor[r.criticity](r.statusShort), r.duration, r.url, humanize.Bytes(uint64(r.size)))
}

// Duration returns the duration of the request
func (r *HTTPRequest) Duration() time.Duration {
	return r.duration
}

// Error returns the error of the request
func (r *HTTPRequest) Error() string {
	var errName string

	switch e := r.err.(type) {
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
			errName = "URL Error"
		}
	case net.Error:
		errName = "Net Error"
	default:
		errName = e.Error()
	}
	return errName
}

// Size returns the size of the request
func (r *HTTPRequest) Size() int64 {
	return r.size
}

// Status returns the status of the request
func (r *HTTPRequest) Status() string {
	return r.status
}

// IsError returns true if the request is an error
func (r *HTTPRequest) IsError() bool {
	return r.err != nil
}

// getURL will get a given URL and return a Request
func getURL(url string) Request {
	// Set the HTTP client
	client := http.DefaultClient
	client.Timeout = time.Duration(timeout) * time.Second
	url = "http://" + url

	var dur time.Duration
	// Initiate the time before the request

	t := time.Now()
	// Do the request
	resp, err := client.Get(url)
	if err != nil {
		dur = time.Since(t)
		return &HTTPRequest{
			url:       url,
			duration:  dur,
			err:       err,
			criticity: Critical,
		}
	}
	defer resp.Body.Close()

	// Read the full body
	len, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		dur = time.Since(t)
		return &HTTPRequest{
			url:       url,
			duration:  dur,
			err:       err,
			criticity: Critical,
		}
	}
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

	return &HTTPRequest{
		url:         url,
		duration:    dur,
		status:      statusText,
		statusShort: strconv.Itoa(resp.StatusCode),
		criticity:   reqCriticity,
		size:        len,
	}
}
