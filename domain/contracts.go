package domain

import (
	"net/http"
	"time"

	"github.com/fllarpy/apm-probe/domain/metrics"
)

// Snapshot is a point-in-time, read-only copy of all metrics.
// This is part of the domain contracts as it defines the data structure
// that application services and infrastructure reporters will work with.
type Snapshot struct {
	ServerEndpoints map[string]metrics.EndpointMetricsSnapshot `json:"server_endpoints"`
	Client          metrics.ClientMetricsSnapshot              `json:"client_metrics"`
	Runtime         metrics.RuntimeMetrics                     `json:"runtime_metrics"`
	Errors          []metrics.ErrorEvent                       `json:"errors"`
	NPlusOneEvents  []metrics.NPlusOneEvent                    `json:"n_plus_one_events"`
}

// StoreReader defines the contract for reading metrics from a store.
type StoreReader interface {
	GetSnapshot() *Snapshot
}

// StoreWriter defines the contract for writing metrics to a store.
type StoreWriter interface {
	AddRequest(path string, duration time.Duration, statusCode int)
	AddClientRequest(duration time.Duration, statusCode int)
	AddError(event metrics.ErrorEvent)
	RecordNPlusOne(path, query string, count int)
	UpdateRuntime()
}

// Store is the combined interface for a metric store.
type Store interface {
	StoreReader
	StoreWriter
}

// Collector defines a component that periodically collects metrics.
type Collector interface {
	Start()
	Stop()
}

// Reporter defines a component that can report metrics, e.g., via an HTTP handler.
type Reporter interface {
	Handler() http.Handler
}
