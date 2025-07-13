package http_reporter

import (
	"encoding/json"
	"net/http"

	"github.com/fllarpy/apm-probe/domain"
)

// NewHandler creates an HTTP handler that serves metrics from the given store.
// It fetches a snapshot of the current metrics and serves it as a JSON response.
func NewHandler(store domain.StoreReader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The snapshot from the store is already in a JSON-serializable format
		// with all the required fields and calculations done.
		snapshot := store.GetSnapshot()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(snapshot); err != nil {
			// If encoding fails, it's a server-side problem.
			http.Error(w, "Failed to encode metrics to JSON", http.StatusInternalServerError)
		}
	})
}
