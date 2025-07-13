package exporter

import (
	"context"
	"testing"
	"time"

	"github.com/fllarpy/apm-probe/storage/inmemory"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type testStore struct {
	inmemory.Store
	requests int
	client   int
	errors   int
}

func (s *testStore) AddRequest(path string, duration time.Duration, statusCode int) {
	s.requests++
	s.Store.AddRequest(path, duration, statusCode)
}
func (s *testStore) AddClientRequest(duration time.Duration, statusCode int) {
	s.client++
	s.Store.AddClientRequest(duration, statusCode)
}
func (s *testStore) AddError(event inmemory.ErrorEvent) { s.errors++; s.Store.AddError(event) }

type mockProfiler struct {
	calls int
}

func (m *mockProfiler) ProfileEndpointIfSlow(path string, duration time.Duration) { m.calls++ }

type mockN1Detector struct {
	calls int
}

func (m *mockN1Detector) ProcessSpan(span sdktrace.ReadOnlySpan) { m.calls++ }

func TestCustomExporter_ExportSpans(t *testing.T) {
	traceID := oteltrace.TraceID{0x01}
	spanID := oteltrace.SpanID{0x01}

	t.Run("processes server span correctly", func(t *testing.T) {
		store := &testStore{}
		profiler := &mockProfiler{}
		detector := &mockN1Detector{}
		exporter, _ := NewCustomExporter(&store.Store, profiler, detector)

		span := &sdktrace.SpanSnapshot{
			SpanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{TraceID: traceID, SpanID: spanID}),
			SpanKind:    oteltrace.SpanKindServer,
			Name:        "/test",
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(10 * time.Millisecond),
		}
		_ = exporter.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{span})

		assert.Equal(t, 1, store.requests, "AddRequest should be called for server spans")
		assert.Equal(t, 1, profiler.calls, "ProfileEndpointIfSlow should be called for server spans")
		assert.Equal(t, 1, detector.calls, "N1Detector.ProcessSpan should be called")
		assert.Equal(t, 0, store.client, "AddClientRequest should not be called")
	})

	t.Run("processes client span correctly", func(t *testing.T) {
		store := &testStore{}
		profiler := &mockProfiler{}
		detector := &mockN1Detector{}
		exporter, _ := NewCustomExporter(&store.Store, profiler, detector)

		span := &sdktrace.SpanSnapshot{
			SpanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{TraceID: traceID, SpanID: spanID}),
			SpanKind:    oteltrace.SpanKindClient,
			Attributes:  []attribute.KeyValue{semconv.DBSystemSqlite},
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(5 * time.Millisecond),
		}
		_ = exporter.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{span})

		assert.Equal(t, 1, store.client, "AddClientRequest should be called for client spans")
		assert.Equal(t, 1, detector.calls, "N1Detector.ProcessSpan should be called")
		assert.Equal(t, 0, store.requests, "AddRequest should not be called")
		assert.Equal(t, 0, profiler.calls, "ProfileEndpointIfSlow should not be called")
	})

	t.Run("processes error span correctly", func(t *testing.T) {
		store := &testStore{}
		profiler := &mockProfiler{}
		detector := &mockN1Detector{}
		exporter, _ := NewCustomExporter(&store.Store, profiler, detector)

		span := &sdktrace.SpanSnapshot{
			SpanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{TraceID: traceID, SpanID: spanID}),
			SpanKind:    oteltrace.SpanKindServer,
			Status:      sdktrace.Status{Code: codes.Error, Description: "something went wrong"},
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(15 * time.Millisecond),
		}
		_ = exporter.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{span})

		assert.Equal(t, 1, store.errors, "AddError should be called for spans with error status")
	})
}
