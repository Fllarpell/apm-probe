package apmhttp

import (
	"net/http"
	"regexp"

	"github.com/fllarpy/apm-probe/domain"
	"github.com/fllarpy/apm-probe/internal/adapters/apmsql"
	"github.com/fllarpy/apm-probe/pkg/config"
)

// A regular expression to find numbers in SQL queries.
// This is a simple approach and might not cover all SQL dialects perfectly.
var sqlNumberRegex = regexp.MustCompile(`\b\d+\b`)

// normalizeQuery replaces numeric literals in a SQL query with a placeholder.
// This helps in detecting N+1 queries where only the ID changes.
// e.g., "SELECT ... WHERE id = 1" and "SELECT ... WHERE id = 2" become "SELECT ... WHERE id = ?".
func normalizeQuery(query string) string {
	return sqlNumberRegex.ReplaceAllString(query, "?")
}

// Middleware is an HTTP middleware that traces requests and detects N+1 query problems.
func Middleware(cfg *config.Config, store domain.StoreWriter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a context for collecting SQL queries for this request.
		ctx := apmsql.WithQueriesContext(r.Context())
		r = r.WithContext(ctx)

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)

		// After the request is done, analyze the collected queries.
		queries := apmsql.QueriesFromContext(ctx)
		// If N+1 detection is enabled (threshold > 0) and the number of queries meets the threshold.
		if cfg.NPlusOneThreshold > 0 && len(queries) >= cfg.NPlusOneThreshold {
			detectNPlusOne(r.URL.Path, queries, store, cfg.NPlusOneThreshold)
		}
	})
}

// detectNPlusOne analyzes a slice of queries and reports potential N+1 problems.
func detectNPlusOne(path string, queries []*apmsql.QueryInfo, store domain.StoreWriter, threshold int) {
	// Simple N+1 detection: count identical query strings.
	// A more advanced version would normalize queries to ignore arguments.
	counts := make(map[string]int)
	for _, q := range queries {
		normalized := normalizeQuery(q.Query)
		counts[normalized]++
	}

	for query, count := range counts {
		if count >= threshold {
			// N+1 detected!
			store.RecordNPlusOne(path, query, count)
		}
	}
}
