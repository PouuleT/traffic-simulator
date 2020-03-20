# Traffic Simulator

[![Build Status](https://github.com/PouuleT/traffic-simulator/workflows/Build/badge.svg?branch=master)](https://github.com/PouuleT/traffic-simulator/workflows/Build/badge.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/PouuleT/traffic-simulator)](https://goreportcard.com/report/github.com/PouuleT/traffic-simulator)

> Simple HTTP and DNS traffic generator written in go

![w000t!](https://w000t.me/2152b71120)

## Usage

```
  -clients int
      number of clients making requests (default 10)
  -requests int
      number of requests to be made by each clients (default 10)
  -seed int
      seed for the random (default 1468538248366626679)
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
```
