package collector

import (
    "sync"
    "time"

    "github.com/fllarpy/apm-probe/domain"
    "github.com/fllarpy/apm-probe/pkg/config"
)

// profilingCollector aggregates runtime metrics from the Store and coordinates
// on-demand CPU profiling. It also tracks per-endpoint cooldowns to avoid
// excessive profiling.
type profilingCollector struct {
    store         domain.Store
    config        *config.Config
    profiler      profiler
    cooldowns     map[string]time.Time
    cooldownsLock sync.Mutex
}

// Start launches a background goroutine that periodically updates runtime
// metrics and delegates profiling checks. It returns a function that stops the
// goroutine.
func Start(store domain.Store, cfg *config.Config) (stop func()) {
    c := &profilingCollector{
        store:     store,
        config:    cfg,
        profiler:  &realProfiler{},
        cooldowns: make(map[string]time.Time),
    }

    done := make(chan struct{})
    var once sync.Once
    ticker := time.NewTicker(cfg.CollectionInterval)

    go func() {
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                c.store.UpdateRuntime()
                c.checkForProblematicEndpoints()
            case <-done:
                return
            }
        }
    }()

    return func() {
        once.Do(func() { close(done) })
    }
}
