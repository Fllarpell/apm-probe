//go:build skip
// +build skip

package nplusone

import (
	"testing"

	"github.com/fllarpy/apm-probe/storage/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// We reuse the real in-memory store for testing purposes.

func TestDetector_ProcessSpan(t *testing.T) {
	cfg := Config{
		Enabled:   true,
		Threshold: 3,
	}
	traceID := oteltrace.TraceID{0x01}
	spanID := oteltrace.SpanID{0x01}

	sqlQuery := "SELECT * FROM users WHERE id = ?"

	t.Run("should not detect with queries below threshold", func(t *testing.T) {
		store := inmemory.NewStore()
		detector := NewDetector(cfg, store)
		require.NotNil(t, detector)

		for i := 0; i < 2; i++ {
			span := createDbSpan(traceID, oteltrace.SpanID{byte(i + 1)}, sqlQuery)
			detector.ProcessSpan(span)
		}

		assert.Equal(t, 0, store.NPlusOneLen(), "RecordNPlusOne should not be called")
	})

	t.Run("should detect when query count reaches threshold", func(t *testing.T) {
		store := inmemory.NewStore()
		detector := NewDetector(cfg, store)
		require.NotNil(t, detector)

		detector.ProcessSpan(createServerSpan(traceID, spanID, "/users"))

		for i := 0; i < 3; i++ {
			span := createDbSpan(traceID, oteltrace.SpanID{byte(i + 2)}, sqlQuery)
			detector.ProcessSpan(span)
		}

		assert.Equal(t, 1, store.NPlusOneLen(), "RecordNPlusOne should be called once")
	})

	t.Run("should report only once per trace", func(t *testing.T) {
		store := inmemory.NewStore()
		detector := NewDetector(cfg, store)
		require.NotNil(t, detector)

		for i := 0; i < 5; i++ {
			span := createDbSpan(traceID, oteltrace.SpanID{byte(i + 1)}, sqlQuery)
			detector.ProcessSpan(span)
		}

		assert.Equal(t, 1, store.NPlusOneLen(), "RecordNPlusOne should only be called once, even if more queries arrive")
	})

	t.Run("should handle different queries in the same trace", func(t *testing.T) {
		store := inmemory.NewStore()
		detector := NewDetector(cfg, store)
		require.NotNil(t, detector)

		detector.ProcessSpan(createDbSpan(traceID, oteltrace.SpanID{0x01}, sqlQuery))
		detector.ProcessSpan(createDbSpan(traceID, oteltrace.SpanID{0x02}, sqlQuery))

		detector.ProcessSpan(createDbSpan(traceID, oteltrace.SpanID{0x03}, "SELECT * FROM products"))
		detector.ProcessSpan(createDbSpan(traceID, oteltrace.SpanID{0x04}, "SELECT * FROM products"))

		assert.Equal(t, 0, store.NPlusOneLen(), "RecordNPlusOne should not be called as no query reached the threshold")
	})
}

func createDbSpan(traceID oteltrace.TraceID, spanID oteltrace.SpanID, query string) sdktrace.ReadOnlySpan {
	return &sdktrace.SpanSnapshot{
		SpanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID: traceID,
			SpanID:  spanID,
		}),
		SpanKind:   oteltrace.SpanKindClient,
		Attributes: []attribute.KeyValue{semconv.DBSystemSqlite, attribute.String("db.statement", query)},
	}
}

func createServerSpan(traceID oteltrace.TraceID, spanID oteltrace.SpanID, path string) sdktrace.ReadOnlySpan {
	return &sdktrace.SpanSnapshot{
		SpanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID: traceID,
			SpanID:  spanID,
		}),
		Name:     path,
		SpanKind: oteltrace.SpanKindServer,
	}
}
