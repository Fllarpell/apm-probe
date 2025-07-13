package apm_probe

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fllarpy/apm-probe/exporter"
	"github.com/fllarpy/apm-probe/nplusone"
	"github.com/fllarpy/apm-probe/profiling"
	"github.com/fllarpy/apm-probe/storage/inmemory"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Probe struct {
	tp *sdktrace.TracerProvider
}

func (p *Probe) Shutdown(ctx context.Context) {
	if err := p.tp.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down tracer provider: %v", err)
	}
}

func NewProbe(ctx context.Context, serviceName string) (*Probe, *inmemory.Store, error) {
	store := inmemory.NewStore()

	profilerCfg := profiling.Config{
		Enabled:          true,
		LatencyThreshold: 500 * time.Millisecond,
		Duration:         10 * time.Second,
		Cooldown:         1 * time.Minute,
	}
	profiler := profiling.NewProfiler(profilerCfg)

	n1detectorCfg := nplusone.Config{
		Enabled:   true,
		Threshold: 5,
	}
	n1detector := nplusone.NewDetector(n1detectorCfg, store)

	customExporter, err := exporter.NewCustomExporter(store, profiler, n1detector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create custom exporter: %w", err)
	}

	res, err := newResource(serviceName, "1.0.0")
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(customExporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	probe := &Probe{
		tp: tp,
	}

	log.Println("APM Probe initialized with custom exporter, profiler, and N+1 detector.")
	return probe, store, nil
}

func newResource(serviceName, serviceVersion string) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
}
