package inmemory

import (
	"runtime"
	"sync"
	"time"

	"github.com/fllarpy/apm-probe/domain"
	"github.com/fllarpy/apm-probe/domain/metrics"
)

const (
	// Default buffer size for events like errors and N+1 queries.
	defaultEventBufferSize = 100
)

// --- Store Implementation ---

// Store is a thread-safe in-memory data store for collecting and serving metrics.
// It implements the domain.Store interface.
var _ domain.Store = (*Store)(nil)

type Store struct {
	mu              sync.RWMutex
	serverEndpoints map[string]*metrics.EndpointMetrics
	client          metrics.ClientMetrics
	runtime         metrics.RuntimeMetrics
	errors          *ringBuffer[metrics.ErrorEvent]
	nPlusOneEvents  *ringBuffer[metrics.NPlusOneEvent]
}

// NewStore creates and initializes a new Store.
func NewStore() *Store {
	return &Store{
		serverEndpoints: make(map[string]*metrics.EndpointMetrics),
		errors:          newRingBuffer[metrics.ErrorEvent](defaultEventBufferSize),
		nPlusOneEvents:  newRingBuffer[metrics.NPlusOneEvent](defaultEventBufferSize),
	}
}

// AddRequest records a new server request.
func (s *Store) AddRequest(path string, duration time.Duration, statusCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint, ok := s.serverEndpoints[path]
	if !ok {
		endpoint = &metrics.EndpointMetrics{}
		s.serverEndpoints[path] = endpoint
	}

	endpoint.TotalRequests++
	endpoint.TotalRequestTime += uint64(duration.Nanoseconds())

	switch {
	case statusCode >= 500:
		endpoint.Status5xx++
	case statusCode >= 400:
		endpoint.Status4xx++
	default:
		endpoint.Status2xx++
	}
}

// AddClientRequest records a new outgoing client request.
func (s *Store) AddClientRequest(duration time.Duration, statusCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.client.TotalRequests++
	s.client.TotalRequestTime += uint64(duration.Nanoseconds())

	switch {
	case statusCode >= 500:
		s.client.Status5xx++
	case statusCode >= 400:
		s.client.Status4xx++
	default:
		s.client.Status2xx++
	}
}

// AddError adds a new error event to the ring buffer.
func (s *Store) AddError(event metrics.ErrorEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors.add(event)
}

// RecordNPlusOne adds a new N+1 event to the ring buffer.
func (s *Store) RecordNPlusOne(path, query string, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := metrics.NPlusOneEvent{
		Timestamp:   time.Now(),
		Path:        path,
		Query:       query,
		Count:       count,
		Description: "N+1 query detected",
	}
	s.nPlusOneEvents.add(event)
}

// UpdateRuntime captures current runtime metrics.
func (s *Store) UpdateRuntime() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.runtime.NumGoroutine = runtime.NumGoroutine()
	s.runtime.MemoryAllocBytes = memStats.Alloc
	s.runtime.MemoryTotalAllocBytes = memStats.TotalAlloc
	s.runtime.MemoryHeapAllocBytes = memStats.HeapAlloc
	s.runtime.MemoryHeapSysBytes = memStats.HeapSys
}

// GetSnapshot returns a read-only copy of the current metrics.
func (s *Store) GetSnapshot() *domain.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := &domain.Snapshot{
		ServerEndpoints: make(map[string]metrics.EndpointMetricsSnapshot),
		Errors:          s.errors.getAll(),
		NPlusOneEvents:  s.nPlusOneEvents.getAll(),
	}

	// Copy server endpoint metrics
	for path, m := range s.serverEndpoints {
		var avgTimeNs uint64
		if m.TotalRequests > 0 {
			avgTimeNs = m.TotalRequestTime / m.TotalRequests
		}
		snapshot.ServerEndpoints[path] = metrics.EndpointMetricsSnapshot{
			TotalRequests:    m.TotalRequests,
			AvgRequestTimeNs: avgTimeNs,
			AvgRequestTime:   time.Duration(avgTimeNs).String(),
			Status2xx:        m.Status2xx,
			Status4xx:        m.Status4xx,
			Status5xx:        m.Status5xx,
		}
	}

	// Copy client metrics
	var avgClientTimeNs uint64
	if s.client.TotalRequests > 0 {
		avgClientTimeNs = s.client.TotalRequestTime / s.client.TotalRequests
	}
	snapshot.Client = metrics.ClientMetricsSnapshot{
		TotalRequests:    s.client.TotalRequests,
		AvgRequestTimeNs: avgClientTimeNs,
		AvgRequestTime:   time.Duration(avgClientTimeNs).String(),
		Status2xx:        s.client.Status2xx,
		Status4xx:        s.client.Status4xx,
		Status5xx:        s.client.Status5xx,
	}

	// Copy runtime metrics
	snapshot.Runtime = s.runtime

	return snapshot
}

// --- Ring Buffer for Events ---

// ringBuffer is a generic, thread-unsafe circular buffer.
// The locking must be handled by the parent (Store).
type ringBuffer[T any] struct {
	buffer []T
	size   int
	start  int
	count  int
}

// newRingBuffer creates a new ring buffer of a given size.
func newRingBuffer[T any](size int) *ringBuffer[T] {
	return &ringBuffer[T]{
		buffer: make([]T, size),
		size:   size,
	}
}

// add inserts an element into the buffer, overwriting the oldest if full.
func (rb *ringBuffer[T]) add(item T) {
	index := (rb.start + rb.count) % rb.size
	rb.buffer[index] = item
	if rb.count < rb.size {
		rb.count++
	} else {
		rb.start = (rb.start + 1) % rb.size
	}
}

// getAll returns all elements in the buffer in order.
func (rb *ringBuffer[T]) getAll() []T {
	if rb.count == 0 {
		return nil
	}
	items := make([]T, rb.count)
	for i := 0; i < rb.count; i++ {
		items[i] = rb.buffer[(rb.start+i)%rb.size]
	}
	return items
}
