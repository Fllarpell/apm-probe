package profiling

import (
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Enabled          bool
	LatencyThreshold time.Duration
	Duration         time.Duration
	Cooldown         time.Duration
}

type Profiler struct {
	config        Config
	cooldowns     map[string]time.Time
	cooldownsLock sync.Mutex
}

func NewProfiler(config Config) *Profiler {
	if !config.Enabled {
		return nil
	}
	log.Println("Initializing on-demand profiler.")
	return &Profiler{
		config:    config,
		cooldowns: make(map[string]time.Time),
	}
}

func (p *Profiler) ProfileEndpointIfSlow(path string, duration time.Duration) {
	if duration < p.config.LatencyThreshold {
		return
	}

	if p.isCoolingDown(path) {
		log.Printf("Profiler: Endpoint '%s' is slow, but is in cooldown.", path)
		return
	}

	log.Printf("Profiler: Endpoint '%s' exceeded latency threshold (%.2fms). Starting CPU profile.", path, float64(duration.Milliseconds()))
	p.setCooldown(path)
	go p.startProfiling(path)
}

func (p *Profiler) startProfiling(path string) {
	sanitizedPath := strings.ReplaceAll(path, "/", "_")
	filename := fmt.Sprintf("%s/profile_%s_%d.pprof", os.TempDir(), sanitizedPath, time.Now().Unix())

	f, err := os.Create(filename)
	if err != nil {
		log.Printf("Profiler: Error creating profile file for '%s': %v", path, err)
		return
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		log.Printf("Profiler: Error starting CPU profile for '%s': %v", path, err)
		return
	}

	time.Sleep(p.config.Duration)
	pprof.StopCPUProfile()

	log.Printf("Profiler: CPU profile for endpoint '%s' completed. Saved to %s", path, filename)
}

func (p *Profiler) isCoolingDown(path string) bool {
	p.cooldownsLock.Lock()
	defer p.cooldownsLock.Unlock()

	if cooldownEnd, exists := p.cooldowns[path]; exists {
		if time.Now().Before(cooldownEnd) {
			return true
		}
		delete(p.cooldowns, path)
	}
	return false
}

func (p *Profiler) setCooldown(path string) {
	p.cooldownsLock.Lock()
	defer p.cooldownsLock.Unlock()

	p.cooldowns[path] = time.Now().Add(p.config.Cooldown)
}
