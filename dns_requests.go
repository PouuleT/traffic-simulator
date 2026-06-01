package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"
)

// DNSGenerator implements the Generator interface
type DNSGenerator struct {
}

func newDNSGenerator(_ config) Generator {
	return &DNSGenerator{}
}

// MakeRequest implements the Generator interface
func (d *DNSGenerator) MakeRequest(ctx context.Context, url string) Request {
	t := time.Now()

	// Extract host from url robustly
	_, host, _, err := parseTargetURL(url)
	if err == nil {
		// Strip port if present
		if h, _, splitErr := net.SplitHostPort(host); splitErr == nil {
			host = h
		}
	} else {
		// Fallback to raw url if parsing fails
		host = url
	}

	// Make the DNS request
	_, err = net.DefaultResolver.LookupHost(ctx, host)
	dur := time.Since(t)

	if err != nil {
		return &DNSRequest{
			duration:  dur,
			url:       url,
			err:       err,
			criticity: Critical,
		}
	}

	return &DNSRequest{
		duration:  dur,
		status:    "OK ",
		criticity: Success,
		url:       url,
	}
}

// DNSRequest represents a request response, with the return code and the duration
type DNSRequest struct {
	status    string
	url       string
	criticity criticityLevel
	duration  time.Duration
	err       error
}

// String will return the string representing the request
func (r *DNSRequest) String() string {
	if r.IsError() {
		return fmt.Sprintf("| %s | %13s | Get %s : %s", red("ERR"), r.duration, r.url, r.Error())
	}
	return fmt.Sprintf("| %s | %13s | Get %s", criticityColor[r.criticity](r.status), r.duration, r.url)
}

// Duration returns the duration of the request
func (r *DNSRequest) Duration() time.Duration {
	return r.duration
}

// Error implements the error interface
func (r *DNSRequest) Error() string {
	if r.err == nil {
		return "<nil>"
	}

	var e *url.Error
	switch {
	case errors.As(r.err, &e) && e.Timeout():
		return "URL timeout"
	case errors.As(r.err, new(*net.DNSError)):
		return "DNS lookup error"
	case errors.As(r.err, new(*net.DNSConfigError)):
		return "DNS config error"
	case errors.As(r.err, new(*net.AddrError)):
		return "Address error"
	case errors.As(r.err, new(*net.OpError)):
		return "Operation error"
	case errors.As(r.err, new(net.Error)):
		return "Network error"
	default:
		return r.err.Error()
	}
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

// URL returns the request target URL
func (r *DNSRequest) URL() string {
	return r.url
}
