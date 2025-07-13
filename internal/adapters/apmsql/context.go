package apmsql

import (
	"context"
	"time"
)

// contextKey is an unexported type for keys defined in this package.
type contextKey struct{}

var queriesKey = contextKey{}

// WithQueriesContext returns a new context capable of storing QueryInfo slices.
// Handlers can attach this to the incoming *http.Request to enable per-request
// aggregation of SQL queries.
func WithQueriesContext(parent context.Context) context.Context {
	return context.WithValue(parent, queriesKey, new([]*QueryInfo))
}

// QueriesFromContext retrieves the collected queries or nil if the context was
// not initialised via WithQueriesContext.
func QueriesFromContext(ctx context.Context) []*QueryInfo {
	p, ok := ctx.Value(queriesKey).(*[]*QueryInfo)
	if !ok || p == nil {
		return nil
	}
	return *p
}

// recordQuery appends information about an executed SQL statement to the slice
// stored in the context. It is used internally by wrapped driver components.
func recordQuery(ctx context.Context, query string, dur time.Duration) {
	p, ok := ctx.Value(queriesKey).(*[]*QueryInfo)
	if !ok || p == nil {
		return
	}
	*p = append(*p, &QueryInfo{Query: query, Duration: dur})
}
