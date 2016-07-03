package main

import (
	"math/rand"
	"net/http"
	"time"
)

// Request represents a requsest reponse, with the return code and the duration
type Request struct {
	status   int
	duration time.Duration
}

func findRandomURL() string {
	return URLs[rand.Intn(len(URLs))]
}

func getURL(url string) (*Request, error) {
	client := http.DefaultClient
	client.Timeout = 3 * time.Second

	t := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	dur := time.Since(t)

	return &Request{
		duration: dur,
		status:   resp.StatusCode,
	}, nil
}
