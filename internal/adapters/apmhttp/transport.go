package apmhttp

import (
	"net/http"
	"time"

	"github.com/fllarpy/apm-probe/domain"
)

// Transport is an http.RoundTripper that measures requests and records them.
type Transport struct {
	// Base is the underlying RoundTripper to execute the request.
	// If nil, http.DefaultTransport is used.
	Base http.RoundTripper

	// Store is the metric store to which metrics will be written.
	store domain.StoreWriter
}

// RoundTrip executes a single HTTP transaction, returning a Response for the request `req`.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	
	// Use the base RoundTripper, or the default if not provided.
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	
	resp, err := base.RoundTrip(req)
	
	duration := time.Since(start)
	
	// If there was an error, we can't get a status code, so we don't record metrics.
	// In a more advanced implementation, we might record this as a failed request.
	if err != nil {
		return nil, err
	}

	t.store.AddClientRequest(duration, resp.StatusCode)
	
	return resp, nil
}

// NewAPMTransport creates a new Transport with the given store.
func NewAPMTransport(base http.RoundTripper, store domain.StoreWriter) *Transport {
	return &Transport{
		Base:  base,
		store: store,
	}
}
