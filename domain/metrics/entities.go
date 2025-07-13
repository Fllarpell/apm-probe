package metrics

import (
	"net/http"
	"time"
)

// --- Data Structures for Metrics ---

// EndpointMetrics holds aggregated metrics for a specific server endpoint.
type EndpointMetrics struct {
	TotalRequests    uint64
	TotalRequestTime uint64 // Stored in nanoseconds for atomic operations
	Status2xx        uint64
	Status4xx        uint64
	Status5xx        uint64
}

// ClientMetrics holds aggregated metrics for all outgoing HTTP client requests.
type ClientMetrics struct {
	TotalRequests    uint64
	TotalRequestTime uint64 // Stored in nanoseconds
	Status2xx        uint64
	Status4xx        uint64
	Status5xx        uint64
}

// RuntimeMetrics holds metrics about the Go runtime.
type RuntimeMetrics struct {
	NumGoroutine          int
	MemoryAllocBytes      uint64
	MemoryTotalAllocBytes uint64
	MemoryHeapAllocBytes  uint64
	MemoryHeapSysBytes    uint64
}

// NPlusOneEvent represents a detected N+1 query problem.
type NPlusOneEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Path        string    `json:"path"`
	Query       string    `json:"query"`
	Count       int       `json:"count"`
	Description string    `json:"description"`
}

// ErrorEvent represents a captured error, typically from a 5xx response.
type ErrorEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Error     string    `json:"error,omitempty"` // Optional error message
}

// NewErrorEvent creates a new ErrorEvent from an HTTP request.
func NewErrorEvent(r *http.Request) ErrorEvent {
	return ErrorEvent{
		Timestamp: time.Now(),
		Method:    r.Method,
		Path:      r.URL.Path,
	}
}


// --- Snapshot Structures (for reporting) ---

// EndpointMetricsSnapshot is a read-only copy of an endpoint's metrics.
type EndpointMetricsSnapshot struct {
	TotalRequests    uint64        `json:"total_requests"`
	AvgRequestTimeNs uint64        `json:"avg_request_time_ns"`
	AvgRequestTime   string        `json:"avg_request_time"`
	Status2xx        uint64        `json:"status_2xx"`
	Status4xx        uint64        `json:"status_4xx"`
	Status5xx        uint64        `json:"status_5xx"`
}

// ClientMetricsSnapshot is a read-only copy of client metrics.
type ClientMetricsSnapshot struct {
	TotalRequests    uint64        `json:"total_requests"`
	AvgRequestTimeNs uint64        `json:"avg_request_time_ns"`
	AvgRequestTime   string        `json:"avg_request_time"`
	Status2xx        uint64        `json:"status_2xx"`
	Status4xx        uint64        `json:"status_4xx"`
	Status5xx        uint64        `json:"status_5xx"`
}
