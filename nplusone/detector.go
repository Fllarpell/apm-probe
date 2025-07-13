package nplusone

import (
	"log"
	"sync"
	"time"

	"github.com/fllarpy/apm-probe/storage/inmemory"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Enabled   bool
	Threshold int
}

type queryInfo struct {
	count     int
	reported  bool
	statement string
}

type traceData struct {
	queries  map[string]*queryInfo
	rootPath string
	lastSeen time.Time
}

type Detector struct {
	config     Config
	store      *inmemory.Store
	traces     map[string]*traceData
	tracesLock sync.Mutex
}

func NewDetector(config Config, store *inmemory.Store) *Detector {
	if !config.Enabled {
		return nil
	}
	log.Println("Initializing N+1 query detector.")
	d := &Detector{
		config: config,
		store:  store,
		traces: make(map[string]*traceData),
	}
	go d.startCleanupRoutine()
	return d
}

func (d *Detector) ProcessSpan(span sdktrace.ReadOnlySpan) {
	traceID := span.SpanContext().TraceID().String()

	d.tracesLock.Lock()
	defer d.tracesLock.Unlock()

	if _, ok := d.traces[traceID]; !ok {
		d.traces[traceID] = &traceData{
			queries:  make(map[string]*queryInfo),
			lastSeen: time.Now(),
		}
	}
	td := d.traces[traceID]
	td.lastSeen = time.Now()

	if span.SpanKind() == trace.SpanKindServer {
		td.rootPath = span.Name()
		return
	}

	var isDbCall bool
	var statement string
	for _, attr := range span.Attributes() {
		if attr.Key == semconv.DBSystemKey {
			isDbCall = true
		}
		if string(attr.Key) == "db.statement" {
			statement = attr.Value.AsString()
		}
	}

	if !isDbCall || statement == "" {
		return
	}

	if _, ok := td.queries[statement]; !ok {
		td.queries[statement] = &queryInfo{statement: statement}
	}
	q := td.queries[statement]
	q.count++

	if q.count >= d.config.Threshold && !q.reported {
		log.Printf("N+1 Detector: Detected problem in trace %s for query: %s", traceID, statement)
		d.store.RecordNPlusOne(td.rootPath, statement, q.count)
		q.reported = true
	}
}

func (d *Detector) startCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		<-ticker.C
		d.cleanupOldTraces()
	}
}

func (d *Detector) cleanupOldTraces() {
	d.tracesLock.Lock()
	defer d.tracesLock.Unlock()

	timeout := 2 * time.Minute
	now := time.Now()
	cleaned := 0
	for traceID, data := range d.traces {
		if now.Sub(data.lastSeen) > timeout {
			delete(d.traces, traceID)
			cleaned++
		}
	}
	if cleaned > 0 {
		log.Printf("N+1 Detector: Cleaned up %d stale traces.", cleaned)
	}
}
