package collector

import (
    "fmt"
    "log"
    "os"
    "strings"
    "time"
)

// checkForProblematicEndpoints checks for endpoints exceeding configured thresholds
// and triggers on-demand CPU profiling when necessary.
func (c *profilingCollector) checkForProblematicEndpoints() {
    if !c.config.ProfilingEnabled {
        return
    }

    snapshot := c.store.GetSnapshot()
    latencyThreshold := uint64(c.config.ProfilingLatencyThreshold.Nanoseconds())
    errorThreshold := uint64(c.config.ProfilingErrorThreshold)

    for path, endpointMetrics := range snapshot.ServerEndpoints {
        triggeredByLatency := endpointMetrics.AvgRequestTimeNs > latencyThreshold
        triggeredByErrors := errorThreshold > 0 && endpointMetrics.Status5xx >= errorThreshold

        if (triggeredByLatency || triggeredByErrors) && !c.isCoolingDown(path) {
            if triggeredByLatency {
                log.Printf("Endpoint '%s' exceeded latency threshold (%.2fms). Starting CPU profile.", path, float64(endpointMetrics.AvgRequestTimeNs)/1e6)
            } else {
                log.Printf("Endpoint '%s' exceeded error threshold (%d errors). Starting CPU profile.", path, endpointMetrics.Status5xx)
            }
            go c.startProfiling(path)
        }
    }
}

// startProfiling runs a CPU profile for a configured duration and writes it to /tmp.
func (c *profilingCollector) startProfiling(path string) {
    c.setCooldown(path)

    sanitizedPath := strings.ReplaceAll(path, "/", "_")
    filename := fmt.Sprintf("/tmp/profile_%s_%d.pprof", sanitizedPath, time.Now().Unix())

    f, err := os.Create(filename)
    if err != nil {
        log.Printf("Error creating profile file for endpoint '%s': %v", path, err)
        return
    }
    defer f.Close()

    if err := c.profiler.StartCPUProfile(f); err != nil {
        log.Printf("Error starting CPU profile for endpoint '%s': %v", path, err)
        return
    }

    time.Sleep(c.config.ProfilingDuration)
    c.profiler.StopCPUProfile()

    log.Printf("CPU profile for endpoint '%s' completed. Saved to %s", path, filename)
}

// cooldown helpers -----------------------------------------------------------

func (c *profilingCollector) isCoolingDown(path string) bool {
    c.cooldownsLock.Lock()
    defer c.cooldownsLock.Unlock()

    if cooldownEnd, exists := c.cooldowns[path]; exists {
        if time.Now().Before(cooldownEnd) {
            return true
        }
        delete(c.cooldowns, path)
    }
    return false
}

func (c *profilingCollector) setCooldown(path string) {
    c.cooldownsLock.Lock()
    defer c.cooldownsLock.Unlock()

    c.cooldowns[path] = time.Now().Add(c.config.ProfilingCooldown)
}
