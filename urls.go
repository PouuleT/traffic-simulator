package main

import (
	"bufio"
	"errors"
	"log"
	"net/url"
	"os"
)

// fillURLs will open the given file and read it to get a list of URLs
func (a *app) fillURLs() error {
	// If no fileName is given, use the defaultURLs variable
	if a.cfg.FileName == "" {
		a.URLs = defaultURLs
		return nil
	}

	file, err := os.Open(a.cfg.FileName)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		u := scanner.Text()
		if _, err := url.Parse(u); err != nil {
			log.Printf("Invalid URL: %q", u)
			continue
		}

		a.URLs = append(a.URLs, u)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if len(a.URLs) == 0 {
		return errors.New("no URL found")
	}

	return nil
}

func (a *app) findRandomURL() string {
	a.URLMutex.Lock()
	defer a.URLMutex.Unlock()
	return a.URLs[a.rng.IntN(len(a.URLs))]
}
