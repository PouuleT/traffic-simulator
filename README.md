# Traffic Simulator

[![Build Status](https://github.com/PouuleT/traffic-simulator/workflows/Build/badge.svg?branch=master)](https://github.com/PouuleT/traffic-simulator/workflows/Build/badge.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/PouuleT/traffic-simulator)](https://goreportcard.com/report/github.com/PouuleT/traffic-simulator)

> HTTP and DNS traffic generator

![w000t!](https://w000t.me/2152b71120)

## Usage

```sh
# Default: TUI mode with 10 clients, 10 requests each
./traffic-simulator

# Custom configuration
./traffic-simulator -clients 5 -requests 20 -wait 500

# DNS mode
./traffic-simulator -type dns

# Plain text output
./traffic-simulator -plain
```

### Flags

```
  -clients int
      number of clients making requests (default 10)
  -requests int
      number of requests to be made by each clients (default 10)
  -seed int
      seed for the random (default: current timestamp)
  -timeout int
      HTTP timeout in seconds (default 3)
  -followRedirect
      follow http redirects or not (default true)
  -type string
      type of requests http/dns (default "http")
  -urlSource string
      optional filepath where to find the URLs
  -wait int
      milliseconds to wait between each requests (default 1000)
  -plain
      use a simplified output (default false)
```

## Building

```sh
go build -o traffic-simulator .
```
