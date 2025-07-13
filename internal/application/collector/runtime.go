package collector

import (
	"time"

	"github.com/fllarpy/apm-probe/domain"
)

// RuntimeCollector is responsible for periodically collecting Go runtime metrics and updating the provided store.
type RuntimeCollector struct {
	store  domain.Store
	interval time.Duration
	ticker  *time.Ticker
}

// NewRuntimeCollector returns a new RuntimeCollector instance.
func NewRuntimeCollector(store domain.Store, interval time.Duration) *RuntimeCollector {
	return &RuntimeCollector{
		store:  store,
		interval: interval,
	}
}

// Start starts the runtime collector.
func (rc *RuntimeCollector) Start() {
	rc.ticker = time.NewTicker(rc.interval)
	go func() {
		defer rc.ticker.Stop()

		for {
			<-rc.ticker.C
			rc.store.UpdateRuntime()
		}
	}()
}
