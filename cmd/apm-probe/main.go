package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/fllarpy/apm-probe/internal/adapters/apmhttp"
	"github.com/fllarpy/apm-probe/internal/application/collector"
	"github.com/fllarpy/apm-probe/internal/ports/http_reporter"
	"github.com/fllarpy/apm-probe/infrastructure/storage/inmemory"
	"github.com/fllarpy/apm-probe/pkg/config"
)

var (
	globalStore *inmemory.Store
	startOnce   sync.Once
)

// init ensures that the agent is initialized only once.
func init() {
	// The actual initialization is deferred to the first call to Middleware or MetricsHandler
	// to ensure configuration is loaded at the right time.
}

// ensureInitialized is a helper function to guarantee the agent is started.
func ensureInitialized() {
	startOnce.Do(func() {
		cfg := config.Load()
		if !cfg.Enabled {
			log.Printf("APM probe disabled")
			return
		}
		globalStore = inmemory.NewStore()
		collector.Start(globalStore, cfg)
	})
}

// Middleware returns a new APM middleware handler.


// MetricsHandler returns an http.Handler that serves the collected metrics.
func MetricsHandler() http.Handler {
	ensureInitialized()
	if globalStore == nil { // Check if initialization was skipped because the agent is disabled.
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/apm/metrics" {
			http_reporter.NewHandler(globalStore).ServeHTTP(w, r)
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	})
}

// NewClient returns an *http.Client that is instrumented to collect metrics.
func NewClient(base *http.Client) *http.Client {
	ensureInitialized()
	if globalStore == nil { // Check if initialization was skipped because the agent is disabled.
		if base != nil {
			return base
		}
		return &http.Client{}
	}

	// Create a new client to avoid modifying the base client's transport.
	client := &http.Client{}
	if base != nil {
		*client = *base
	}

	baseTransport := client.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}

	// Set our custom transport.
	client.Transport = apmhttp.NewAPMTransport(baseTransport, globalStore)

	return client
}
