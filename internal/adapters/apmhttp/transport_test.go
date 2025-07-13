package apmhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fllarpy/apm-probe/infrastructure/storage/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransport_RoundTrip(t *testing.T) {
	testCases := []struct {
		name        string
		statusCode  int
		expected2xx uint64
		expected4xx uint64
		expected5xx uint64
	}{
		{"OK", http.StatusOK, 1, 0, 0},
		{"Not Found", http.StatusNotFound, 0, 1, 0},
		{"Internal Server Error", http.StatusInternalServerError, 0, 0, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Setup
			store := inmemory.NewStore()

			// Create a mock server that returns the configured status code.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			// Create a client with our custom transport.
			transport := NewAPMTransport(nil, store)
			client := &http.Client{Transport: transport}

			// 2. Execution
			resp, err := client.Get(server.URL)
			require.NoError(t, err, "client.Get should not return an error")
			require.Equal(t, tc.statusCode, resp.StatusCode, "response status code should match expected")

			// 3. Verification
			snapshot := store.GetSnapshot()

			// Check that the client metrics were recorded correctly.
			clientMetrics := snapshot.Client
			assert.Equal(t, uint64(1), clientMetrics.TotalRequests, "TotalClientRequests should be 1")
			assert.Equal(t, tc.expected2xx, clientMetrics.Status2xx, "ClientStatus2xx should match expected")
			assert.Equal(t, tc.expected4xx, clientMetrics.Status4xx, "ClientStatus4xx should match expected")
			assert.Equal(t, tc.expected5xx, clientMetrics.Status5xx, "ClientStatus5xx should match expected")
			assert.Greater(t, clientMetrics.AvgRequestTimeNs, uint64(0), "AvgRequestTimeNs should be recorded for a single request")

			// Check that no server metrics were recorded by this transport.
			assert.Empty(t, snapshot.ServerEndpoints, "Transport should not record any server-side metrics")
		})
	}
}
