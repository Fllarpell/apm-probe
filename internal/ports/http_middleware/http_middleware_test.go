package http_middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fllarpy/apm-probe/infrastructure/storage/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPMMiddleware(t *testing.T) {
	t.Run("Agent Enabled", func(t *testing.T) {
		// This is not strictly necessary as true is the default, but it's good for clarity.
		t.Setenv("APM_ENABLED", "true")

		testCases := []struct {
			name               string
			statusCode         int
			expected2xx        uint64
			expected4xx        uint64
			expected5xx        uint64
			expectedErrorCount int
		}{
			{"OK", http.StatusOK, 1, 0, 0, 0},
			{"Not Found", http.StatusNotFound, 0, 1, 0, 0},
			{"Internal Server Error", http.StatusInternalServerError, 0, 0, 1, 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Setup
								store := inmemory.NewStore()
				requestPath := "/test/path"
				testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.statusCode)
				})
				middleware := APMMiddleware(store)
				wrappedHandler := middleware(testHandler)

				// Execution
				req := httptest.NewRequest("GET", requestPath, nil)
				rr := httptest.NewRecorder()
				wrappedHandler.ServeHTTP(rr, req)

				// Verification
				snapshot := store.GetSnapshot()
				require.Contains(t, snapshot.ServerEndpoints, requestPath, "metrics should be recorded for the correct path")

				endpointMetrics := snapshot.ServerEndpoints[requestPath]
				assert.Equal(t, uint64(1), endpointMetrics.TotalRequests, "TotalRequests should be 1")
				assert.Equal(t, tc.expected2xx, endpointMetrics.Status2xx, "2xx status codes should match")
				assert.Equal(t, tc.expected4xx, endpointMetrics.Status4xx, "4xx status codes should match")
				assert.Equal(t, tc.expected5xx, endpointMetrics.Status5xx, "5xx status codes should match")
				assert.Len(t, snapshot.Errors, tc.expectedErrorCount, "error count should match")
			})
		}
	})

	t.Run("Agent Disabled", func(t *testing.T) {
		t.Setenv("APM_ENABLED", "false")

		// Setup
						store := inmemory.NewStore()
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		middleware := APMMiddleware(store)
		wrappedHandler := middleware(testHandler)

		// Execution
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)

		// Verification
		snapshot := store.GetSnapshot()
		assert.Empty(t, snapshot.ServerEndpoints, "no server metrics should be recorded when agent is disabled")
		assert.Empty(t, snapshot.Errors, "no errors should be recorded when agent is disabled")
	})
}
