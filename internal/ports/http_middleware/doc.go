// Package http_middleware provides HTTP middleware for collecting metrics
// and detecting N+1 query issues in HTTP handlers. It wraps HTTP handlers
// to track request timing, status codes, and SQL query patterns.
//
// The middleware is designed to be used with the standard library's
// net/http package and integrates with the application's metrics store.
package http_middleware
