package http_reporter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fllarpy/apm-probe/domain"
	"github.com/fllarpy/apm-probe/infrastructure/storage/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsHandler(t *testing.T) {
	// 1. Setup: Create a store and populate it with metrics for multiple endpoints.
	store := inmemory.NewStore()
	path1 := "/api/v1/users"
	path2 := "/api/v2/posts"

	// Metrics for path1
	store.AddRequest(path1, 100*time.Millisecond, http.StatusOK)       // 200
	store.AddRequest(path1, 150*time.Millisecond, http.StatusCreated) // 201

	// Metrics for path2
	store.AddRequest(path2, 200*time.Millisecond, http.StatusNotFound)       // 404
	store.AddRequest(path2, 300*time.Millisecond, http.StatusInternalServerError) // 500

	// Add some client and runtime metrics for completeness
	store.AddClientRequest(50*time.Millisecond, 200)
	store.UpdateRuntime()

	handler := NewHandler(store)

	// 2. Execution: Make a request to the metrics handler.
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// 3. Verification: Check the response.
	require.Equal(t, http.StatusOK, rr.Code, "handler should return status OK")

	// Unmarshal the response into our primary metrics snapshot struct.
		var snapshot domain.Snapshot
	err := json.Unmarshal(rr.Body.Bytes(), &snapshot)
	require.NoError(t, err, "Failed to unmarshal response body")

	// --- Assertions for Server-side Endpoint Metrics ---
	require.Len(t, snapshot.ServerEndpoints, 2, "Should be metrics for two endpoints")

	// Check metrics for path1
	require.Contains(t, snapshot.ServerEndpoints, path1)
	snap1 := snapshot.ServerEndpoints[path1]
	assert.Equal(t, uint64(2), snap1.TotalRequests)
	assert.Equal(t, uint64(2), snap1.Status2xx)
	assert.Equal(t, uint64(0), snap1.Status4xx)
	assert.Equal(t, uint64(0), snap1.Status5xx)
	expectedAvg1 := uint64((100*time.Millisecond + 150*time.Millisecond) / 2)
	assert.Equal(t, expectedAvg1, snap1.AvgRequestTimeNs)

	// Check metrics for path2
	require.Contains(t, snapshot.ServerEndpoints, path2)
	snap2 := snapshot.ServerEndpoints[path2]
	assert.Equal(t, uint64(2), snap2.TotalRequests)
	assert.Equal(t, uint64(0), snap2.Status2xx)
	assert.Equal(t, uint64(1), snap2.Status4xx)
	assert.Equal(t, uint64(1), snap2.Status5xx)
	expectedAvg2 := uint64((200*time.Millisecond + 300*time.Millisecond) / 2)
	assert.Equal(t, expectedAvg2, snap2.AvgRequestTimeNs)

	// --- Assertions for Client and Runtime Metrics ---
	assert.Equal(t, uint64(1), snapshot.Client.TotalRequests)
	assert.Greater(t, snapshot.Runtime.NumGoroutine, 0)
	assert.Greater(t, snapshot.Runtime.MemoryAllocBytes, uint64(0))
}
