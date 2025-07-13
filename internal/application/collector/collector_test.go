package collector

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/fllarpy/apm-probe/infrastructure/storage/inmemory"
	"github.com/fllarpy/apm-probe/pkg/config"
	"github.com/stretchr/testify/assert"
)

// mockProfiler allows us to intercept profiling calls for testing.
// It implements the profiler interface.
type mockProfiler struct {
	startCalled chan bool
}

func (mp *mockProfiler) StartCPUProfile(w io.Writer) error {
	// In the mock, we just signal that the method was called.
	mp.startCalled <- true
	return nil
}

func (mp *mockProfiler) StopCPUProfile() {}

func TestCheckForProblematicEndpoints(t *testing.T) {
	store := inmemory.NewStore()
	mockProfiler := &mockProfiler{startCalled: make(chan bool, 1)}

	cfg := &config.Config{
		ProfilingEnabled:          true,
		ProfilingLatencyThreshold: 100 * time.Millisecond,
		ProfilingErrorThreshold:   5,
		ProfilingDuration:         10 * time.Millisecond, // Short duration for test
		ProfilingCooldown:         300 * time.Millisecond,
	}

	// Create the collector and inject our mock profiler.
	collector := &profilingCollector{
		store:     store,
		config:    cfg,
		profiler:  mockProfiler,
		cooldowns: make(map[string]time.Time),
	}

	// 1. Test case: one endpoint is slow, one is fast.
	store.AddRequest("/api/slow", 150*time.Millisecond, http.StatusInternalServerError)
	store.AddRequest("/api/fast", 50*time.Millisecond, http.StatusOK)
	store.AddRequest("/api/fast", 50*time.Millisecond, http.StatusOK)

	collector.checkForProblematicEndpoints()

	// Wait for the profiler to be called for the slow endpoint.
	select {
	case <-mockProfiler.startCalled:
		// Success!
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timed out waiting for profiling to start")
	}

	assert.True(t, collector.isCoolingDown("/api/slow"), "/api/slow should be in cooldown")
	assert.False(t, collector.isCoolingDown("/api/fast"), "/api/fast should not be in cooldown")

	// 2. Test case: slow endpoint is now in cooldown, should not be profiled again.
	collector.checkForProblematicEndpoints()
	select {
	case <-mockProfiler.startCalled:
		t.Fatal("Profiling started, but it should be in cooldown")
	default:
		// Good, nothing was profiled.
	}

	// 3. Test case: cooldown expires.
	time.Sleep(cfg.ProfilingCooldown)
	assert.False(t, collector.isCoolingDown("/api/slow"), "/api/slow cooldown should have expired")

	// It should be profiled again due to latency.
	collector.checkForProblematicEndpoints()
	select {
	case <-mockProfiler.startCalled:
		// Success!
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timed out waiting for profiling to start after cooldown")
	}
	assert.True(t, collector.isCoolingDown("/api/slow"), "/api/slow should be in cooldown again")

	// 4. Test case: trigger by error count.
	store.AddRequest("/api/error", 50*time.Millisecond, http.StatusInternalServerError)
	store.AddRequest("/api/error", 50*time.Millisecond, http.StatusInternalServerError)
	store.AddRequest("/api/error", 50*time.Millisecond, http.StatusInternalServerError)
	store.AddRequest("/api/error", 50*time.Millisecond, http.StatusInternalServerError)
	store.AddRequest("/api/error", 50*time.Millisecond, http.StatusInternalServerError)
	store.AddRequest("/api/error", 50*time.Millisecond, http.StatusInternalServerError)

	// Wait for the previous cooldown to expire to avoid interference.
	time.Sleep(cfg.ProfilingCooldown)
	assert.False(t, collector.isCoolingDown("/api/slow"), "/api/slow cooldown should have expired")

	collector.checkForProblematicEndpoints()

	select {
	case <-mockProfiler.startCalled:
		// Success!
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timed out waiting for profiling to start on error threshold")
	}
	assert.True(t, collector.isCoolingDown("/api/error"), "/api/error should be in cooldown")
}
