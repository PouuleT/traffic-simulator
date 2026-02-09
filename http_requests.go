package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
)

// HTTPRequest represents a request response, with the return code and the duration
type HTTPRequest struct {
	status           string
	statusShort      string
	url              string
	criticity        criticityLevel
	duration         time.Duration
	err              error
	size             int64
	responseTimeline *ResponseTimeline
}

type ResponseTimeline struct {
	DNSLookup              time.Duration
	TCPConnection          time.Duration
	EstablishingConnection time.Duration
	ServerProcessing       time.Duration
	ContentTransfer        time.Duration
}

// HTTPGenerator implements the Generator interface
type HTTPGenerator struct {
	client *http.Client
}

func newHTTPGenerator(cfg config) Generator {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			// Check if we need to follow redirect or no
			if cfg.FollowHTTPRedirect {
				return nil
			}
			return http.ErrUseLastResponse
		},
	}
	return &HTTPGenerator{
		client: client,
	}
}

// MakeRequest implements the Generator interface
func (h *HTTPGenerator) MakeRequest(ctx context.Context, url string) Request {
	url = "http://" + url
	var dnsStart, dnsDone, connectStart, connectDone, gotConn, gotByte time.Time

	// Do the request
	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:  func(_ httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		ConnectStart: func(_, _ string) {
			if dnsDone.IsZero() {
				// connecting to IP
				dnsDone = time.Now()
			}
			connectStart = time.Now()
		},
		ConnectDone: func(_, addr string, err error) {
			if err != nil {
				log.Printf("unable to connect to host %v: %v", addr, err)
			}
			connectDone = time.Now()
		},
		GotConn:              func(_ httptrace.GotConnInfo) { gotConn = time.Now() },
		GotFirstResponseByte: func() { gotByte = time.Now() },
	}

	// Initiate the time before the request
	t := time.Now()

	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), "GET", url, nil)
	if err != nil {
		dur := time.Since(t)
		return &HTTPRequest{
			url:       url,
			duration:  dur,
			err:       err,
			criticity: Critical,
		}
	}

	resp, err := h.client.Do(req)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		dur := time.Since(t)
		return &HTTPRequest{
			url:       url,
			duration:  dur,
			err:       err,
			criticity: Critical,
		}
	}

	// Read the full body
	length, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		dur := time.Since(t)
		return &HTTPRequest{
			url:       url,
			duration:  dur,
			err:       err,
			criticity: Critical,
			size:      length,
		}
	}
	// Record the duration of the request
	allDone := time.Now()
	dur := time.Since(t)

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

	// Calculate timeline durations, handling zero times for reused connections
	var dnsLookup, tcpConnection, establishingConnection, serverProcessing, contentTransfer time.Duration

	if !dnsStart.IsZero() && !dnsDone.IsZero() {
		dnsLookup = dnsDone.Sub(dnsStart)
	}
	if !connectStart.IsZero() && !connectDone.IsZero() {
		tcpConnection = connectDone.Sub(connectStart)
	}
	if !connectDone.IsZero() && !gotConn.IsZero() && gotConn.After(connectDone) {
		establishingConnection = gotConn.Sub(connectDone)
	}
	if !gotConn.IsZero() && !gotByte.IsZero() {
		serverProcessing = gotByte.Sub(gotConn)
	}
	if !gotByte.IsZero() && !allDone.IsZero() {
		contentTransfer = allDone.Sub(gotByte)
	}

	responseTimeline := ResponseTimeline{
		DNSLookup:              dnsLookup,
		TCPConnection:          tcpConnection,
		EstablishingConnection: establishingConnection,
		ServerProcessing:       serverProcessing,
		ContentTransfer:        contentTransfer,
	}

	return &HTTPRequest{
		url:              url,
		duration:         dur,
		status:           statusText,
		statusShort:      strconv.Itoa(resp.StatusCode),
		criticity:        reqCriticity,
		size:             length,
		responseTimeline: &responseTimeline,
	}
}

// String will return the string representing the request
func (r HTTPRequest) String() string {
	if r.IsError() {
		return fmt.Sprintf("| %s | %13s | Get %s : %s ( %s )", red("ERR"), r.duration, r.url, r.Error(), humanize.Bytes(uint64(r.size)))
	}
	return fmt.Sprintf("| %s | %13s | Get %s ( %s )", criticityColor[r.criticity](r.statusShort), r.duration, r.url, humanize.Bytes(uint64(r.size)))
}

// Duration returns the duration of the request
func (r HTTPRequest) Duration() time.Duration {
	return r.duration
}

// Error returns the error of the request
func (r HTTPRequest) Error() string {
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
func (r HTTPRequest) Size() int64 {
	return r.size
}

// Status returns the status of the request
func (r HTTPRequest) Status() string {
	return r.status
}

// IsError returns true if the request is an error
func (r HTTPRequest) IsError() bool {
	return r.err != nil
}
