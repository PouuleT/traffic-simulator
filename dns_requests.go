package main

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

// DNSRequest represents a requsest reponse, with the return code and the duration
type DNSRequest struct {
	status      string
	statusShort string
	url         string
	criticity   criticityLevel
	duration    time.Duration
	err         error
}

// String will return the string representing the request
func (r *DNSRequest) String() string {
	if r.IsError() {
		return fmt.Sprintf("| %s | %12s | Get %s : %s", red("ERR"), r.duration, r.url, r.Error())
	}
	return fmt.Sprintf("| %s | %12s | Get %s", criticityColor[r.criticity](r.status), r.duration, r.url)
}

// Duration returns the duration of the request
func (r *DNSRequest) Duration() time.Duration {
	return r.duration
}

// Error implements the error interface
func (r *DNSRequest) Error() string {
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
func (r *DNSRequest) Size() int64 {
	return 0
}

// Status returns the status of the request
func (r *DNSRequest) Status() string {
	return r.status
}

// IsError returns true if the request is an error
func (r *DNSRequest) IsError() bool {
	return r.err != nil
}

// lookupURL will make a DNS request on a given URL and return a Request
func lookupURL(url string) Request {
	var dur time.Duration
	t := time.Now()
	// Make the DNS request
	_, err := net.LookupHost(url)
	if err != nil {
		dur = time.Since(t)
		return &DNSRequest{
			duration:  dur,
			url:       url,
			err:       err,
			criticity: Critical,
		}
	}

	// Record the duration of the request
	dur = time.Since(t)

	return &DNSRequest{
		duration:  dur,
		status:    "OK ",
		criticity: Success,
		url:       url,
	}
}
