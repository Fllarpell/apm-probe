// Package http_reporter provides an HTTP handler for exposing collected metrics
// in JSON format. It's intended to be used by monitoring systems to scrape
// the current state of the application's metrics.
//
// The package implements the standard http.Handler interface and can be
// mounted on any HTTP router or used with the standard library's http package.
package http_reporter
