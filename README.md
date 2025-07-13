# Go APM Probe

A lightweight, embeddable Application Performance Monitoring (APM) agent for Go applications.

This probe automatically collects detailed metrics from HTTP servers and clients, captures error events, and gathers Go runtime metrics with minimal overhead.

## Features

- **HTTP Server Metrics**: Automatically instruments incoming HTTP requests to track request counts, latency, and status codes (2xx, 4xx, 5xx).
- **HTTP Client Metrics**: Instruments outgoing HTTP requests made with an instrumented `*http.Client`.
- **Error Collection**: Captures details of server-side 5xx errors, including the request path, method, and timestamp.
- **Runtime Metrics**: Periodically collects Go runtime statistics, such as the number of active goroutines and memory allocation details (`Alloc`, `TotalAlloc`, `HeapAlloc`, `HeapSys`).
- **Configurable Metrics Endpoint**: Exposes all collected metrics via a JSON endpoint (default: `/debug/apm`).
- **Enable/Disable via Environment**: Can be easily enabled or disabled globally.

## Installation

```bash
go get github.com/fllarpy/apm-probe
```

## Quick Start

To add the APM probe to your Go web application, you need to:

1.  Wrap your main HTTP handler with the `apm_probe.Middleware()`.
2.  Register the `apm_probe.MetricsHandler()` to expose the metrics endpoint.

Here is a simple example:

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/fllarpy/apm-probe"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, world!")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", helloHandler)

	// Register the metrics handler
	metricsHandler := apm_probe.MetricsHandler()
	if metricsHandler != nil {
		mux.Handle("/debug/apm", metricsHandler)
	}

	// Wrap the entire mux with the APM middleware
	app := apm_probe.Middleware()(mux)

	log.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", app))
}
```

## Configuration

The probe is configured using environment variables:

| Variable                  | Description                                                              | Default         |
| ------------------------- | ------------------------------------------------------------------------ | --------------- |
| `APM_ENABLED`             | Set to `true` or `false` to enable or disable the probe.                 | `true`          |
| `APM_DEBUG_ENDPOINT`      | The path for the metrics HTTP endpoint.                                  | `/debug/apm`    |
| `APM_COLLECTION_INTERVAL` | The interval for collecting runtime metrics (e.g., `5s`, `1m`).          | `10s`           |

## Instrumented HTTP Client

To monitor outgoing HTTP requests, use the `apm_probe.NewClient()` constructor:

```go
// Create a new instrumented client
client := apm_probe.NewClient(nil) // Pass an existing client or nil to use defaults

// Make requests as usual
resp, err := client.Get("https://example.com")
// ...
```
