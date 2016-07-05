package main

import (
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

// Request represents a requsest reponse, with the return code and the duration
type Request struct {
	status   int
	duration time.Duration
	err      error
}

// findRandomURL will return a random URL
func findRandomURL() string {
	return URLs[rand.Intn(len(URLs))]
}

// getURL will get a given URL and return a *Request
func getURL(url string) *Request {
	client := http.DefaultClient
	client.Timeout = 3 * time.Second

	var dur time.Duration
	t := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		dur = time.Since(t)
		return &Request{
			duration: dur,
			status:   0,
			err:      err,
		}
	}
	defer resp.Body.Close()

	// Read the full body
	io.Copy(ioutil.Discard, resp.Body)
	dur = time.Since(t)

	return &Request{
		duration: dur,
		status:   resp.StatusCode,
	}
}
