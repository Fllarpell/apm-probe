package http

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewMiddleware(handler http.Handler, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation)
} 
