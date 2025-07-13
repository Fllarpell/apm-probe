// Package apmhttp provides adapters for instrumenting HTTP clients and servers
// with APM capabilities. It includes middleware for HTTP handlers and a
// transport wrapper for HTTP clients to collect request/response metrics
// and trace distributed transactions.
//
// The package is designed to work with the standard library's net/http
// package and follows Go's idiomatic patterns for HTTP middleware.
package apmhttp
