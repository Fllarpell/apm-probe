package exporter

import (
	"context"
	"log"
	"time"

	"github.com/fllarpy/apm-probe/storage/inmemory"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Profiler is a very small interface used by the exporter. It allows test
// suites to inject lightweight mocks without depending on the concrete
// implementation from the profiling package.
type Profiler interface {
	// ProfileEndpointIfSlow profiles an endpoint when its latency exceeds a
	// threshold. The real implementation is provided by profiling.Profiler.
	ProfileEndpointIfSlow(path string, duration time.Duration)
}

// N1Detector is the minimal interface the exporter relies on for detecting
// N+1-запросы.  The concrete implementation lives in nplusone.Detector but
// tests can substitute it with a stub.
type N1Detector interface {
	ProcessSpan(span sdktrace.ReadOnlySpan)
}

type CustomExporter struct {
	store      *inmemory.Store
	profiler   Profiler
	n1detector N1Detector
}

func NewCustomExporter(store *inmemory.Store, profiler Profiler, n1detector N1Detector) (*CustomExporter, error) {
	log.Println("Initializing custom exporter.")
	return &CustomExporter{
		store:      store,
		profiler:   profiler,
		n1detector: n1detector,
	}, nil
}

func (e *CustomExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		if e.n1detector != nil {
			e.n1detector.ProcessSpan(span)
		}

		switch span.SpanKind() {
		case trace.SpanKindServer:
			e.processServerSpan(span)
		case trace.SpanKindClient:
			e.processClientSpan(span)
		}
	}
	return nil
}

func (e *CustomExporter) Shutdown(ctx context.Context) error {
	log.Println("Custom exporter shut down.")
	return nil
}

func (e *CustomExporter) processServerSpan(span sdktrace.ReadOnlySpan) {
	duration := span.EndTime().Sub(span.StartTime())
	path := span.Name()

	var statusCode int
	var errorMsg string
	hasError := span.Status().Code == codes.Error

	for _, attr := range span.Attributes() {
		if string(attr.Key) == "http.status_code" {
			statusCode = int(attr.Value.AsInt64())
		}
		if string(attr.Key) == "exception.message" {
			errorMsg = attr.Value.AsString()
		}
	}

	if statusCode >= 500 {
		hasError = true
	}

	log.Printf("CustomExporter: Processed SERVER span: %s, Duration: %s, Status: %d", path, duration, statusCode)
	e.store.AddRequest(path, duration, statusCode)

	if hasError {
		e.store.AddError(inmemory.ErrorEvent{
			Timestamp: span.EndTime(),
			Method:    "",
			Path:      path,
			Error:     errorMsg,
		})
	}

	if e.profiler != nil {
		e.profiler.ProfileEndpointIfSlow(path, duration)
	}
}

func (e *CustomExporter) processClientSpan(span sdktrace.ReadOnlySpan) {
	duration := span.EndTime().Sub(span.StartTime())
	var hasError bool

	if span.Status().Code == codes.Error {
		hasError = true
	}

	for _, attr := range span.Attributes() {
		if attr.Key == semconv.DBSystemKey {
			log.Printf("CustomExporter: Processed CLIENT span (db): %s, Duration: %s", span.Name(), duration)
			e.store.AddClientRequest(duration, 0)
			break
		}
	}

	if hasError {
		log.Printf("CustomExporter: Client span had an error: %s", span.Name())
	}
}
