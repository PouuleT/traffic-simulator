package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
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
			errName = fmt.Sprintf("URL Error: %s", e.Error())
		}
	case net.Error:
		errName = "Net Error"
	default:
		errName = e.Error()
	}
	return errName
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

// getURL will get a given URL and return a Request
func oldgetURL(url string) Request {
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
			size:      len,
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

// getURL will get a given URL and return a Request
func getURL(url string) Request {
	var dnsStart, dnsDone, connectStart, connectDone, gotConn, gotByte time.Time
	url = "http://" + url

	var dur time.Duration
	// Initiate the time before the request

	t := time.Now()

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
		ConnectDone: func(net, addr string, err error) {
			if err != nil {
				log.Printf("unable to connect to host %v: %v", addr, err)
				return
			}
			connectDone = time.Now()
		},
		GotConn:              func(_ httptrace.GotConnInfo) { gotConn = time.Now() },
		GotFirstResponseByte: func() { gotByte = time.Now() },
	}

	b := strings.NewReader("")
	req, err := http.NewRequest("GET", url, b)
	if err != nil {
		dur = time.Since(t)
		return &HTTPRequest{
			url:       url,
			duration:  dur,
			err:       err,
			criticity: Critical,
		}
	}

	req = req.WithContext(httptrace.WithClientTrace(context.Background(), trace))

	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Check if we need to follow redirect or no
			if followHttpRedirect {
				return nil
			}
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
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
			size:      len,
		}
	}
	// Record the duration of the request
	allDone := time.Now()
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

	responseTimeline := ResponseTimeline{
		DNSLookup:              dnsDone.Sub(dnsStart),
		TCPConnection:          connectDone.Sub(connectStart),
		EstablishingConnection: gotConn.Sub(connectDone),
		ServerProcessing:       gotByte.Sub(gotConn),
		ContentTransfer:        allDone.Sub(gotByte),
	}

	return &HTTPRequest{
		url:              url,
		duration:         dur,
		status:           statusText,
		statusShort:      strconv.Itoa(resp.StatusCode),
		criticity:        reqCriticity,
		size:             len,
		responseTimeline: &responseTimeline,
	}
}
