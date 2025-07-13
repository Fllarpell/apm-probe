package http_middleware

import (
	"net/http"
	"time"

	"github.com/fllarpy/apm-probe/internal/adapters/apmhttp"
	"github.com/fllarpy/apm-probe/pkg/config"
	"github.com/fllarpy/apm-probe/domain"
	"github.com/fllarpy/apm-probe/domain/metrics"
)

// responseWriter is a wrapper around http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// APMMiddleware creates a new HTTP middleware that collects request metrics and
// detects N+1 query problems. It returns a function that takes an http.Handler
// and returns an http.Handler, suitable for use with frameworks like chi.
func APMMiddleware(store domain.StoreWriter) func(http.Handler) http.Handler {
	cfg := config.Load()
	if !cfg.Enabled || store == nil {
		// If disabled, return a no-op middleware.
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Return the actual middleware function.
	return func(next http.Handler) http.Handler {
		// This handler will be wrapped by the N+1 detector.
		handlerWithMetrics := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			duration := time.Since(start)
			store.AddRequest(r.URL.Path, duration, rw.statusCode)

			// If the status code is 5xx, record it as an error event.
			if rw.statusCode >= 500 {
								event := metrics.NewErrorEvent(r)
				store.AddError(event)
			}
		})

		// The N+1 detector wraps our metrics handler.
		return apmhttp.Middleware(cfg, store, handlerWithMetrics)
	}
}
