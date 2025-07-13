package inmemory

import (
	"sync"
	"time"
)

// ErrorEvent represents an occurred error for a request.
// It is a trimmed-down replacement for the former domain/metrics.ErrorEvent.
type ErrorEvent struct {
	Timestamp time.Time
	Method    string
	Path      string
	Error     string
}

// Snapshot is a very lightweight representation of the current aggregated data.
// Only the tests need this at the moment.
type Snapshot struct {
	TotalRequests int
	TotalErrors   int
}

// Store is a minimal, goroutine-safe in-memory implementation that collects
// basic statistics required by CustomExporter, Profiler and N+1 detector.
type Store struct {
	mu sync.Mutex

	requests       []requestEntry
	clientRequests []clientEntry
	errors         []ErrorEvent

	nPlusOneEvents []nPlusOneEntry
}

type requestEntry struct {
	Path       string
	Duration   time.Duration
	StatusCode int
}

type clientEntry struct {
	Duration   time.Duration
	StatusCode int
}

type nPlusOneEntry struct {
	Path      string
	Statement string
	Count     int
}

// NewStore returns a ready-to-use Store instance.
func NewStore() *Store {
	return &Store{}
}

// AddRequest records a server request.
func (s *Store) AddRequest(path string, duration time.Duration, statusCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, requestEntry{path, duration, statusCode})
}

// AddClientRequest records a downstream client request (e.g., DB query).
func (s *Store) AddClientRequest(duration time.Duration, statusCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clientRequests = append(s.clientRequests, clientEntry{duration, statusCode})
}

// AddError records an application error.
func (s *Store) AddError(event ErrorEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors = append(s.errors, event)
}

// RecordNPlusOne registers a detected N+1 query problem.
func (s *Store) RecordNPlusOne(path, statement string, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nPlusOneEvents = append(s.nPlusOneEvents, nPlusOneEntry{path, statement, count})
}

// NPlusOneLen returns how many N+1 events were recorded. This helper is used
// exclusively in unit tests.
func (s *Store) NPlusOneLen() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.nPlusOneEvents)
}

// UpdateRuntime is a stub kept for backward compatibility. It can later be
// wired to collect runtime stats (GC, mem, etc.).
func (s *Store) UpdateRuntime() {}

// GetSnapshot returns a very simple snapshot – sufficient for unit tests.
func (s *Store) GetSnapshot() *Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &Snapshot{
		TotalRequests: len(s.requests),
		TotalErrors:   len(s.errors),
	}
}
