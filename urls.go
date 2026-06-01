package main

import (
	"bufio"
	"errors"
	"log"
	"net/url"
	"os"
	"strings"
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
		u := strings.TrimSpace(scanner.Text())
		if u == "" {
			continue
		}
		if _, _, _, err := parseTargetURL(u); err != nil {
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

// parseTargetURL parses raw input strings (with or without schemes) and extracts scheme, host, and path.
func parseTargetURL(raw string) (scheme, host, path string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", errors.New("empty URL")
	}

	hasScheme := strings.HasPrefix(strings.ToLower(raw), "http://") ||
		strings.HasPrefix(strings.ToLower(raw), "https://")

	parseStr := raw
	if !hasScheme {
		parseStr = "http://" + raw
	}

	u, err := url.Parse(parseStr)
	if err != nil {
		return "", "", "", err
	}

	scheme = u.Scheme
	host = u.Host
	path = u.Path
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		path += "#" + u.Fragment
	}

	if !hasScheme {
		scheme = ""
	}
	return scheme, host, path, nil
}
