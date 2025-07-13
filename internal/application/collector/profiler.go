package collector

import (
    "io"
    "runtime/pprof"
)

// profiler abstracts CPU profiling so tests can mock it.
type profiler interface {
    StartCPUProfile(w io.Writer) error
    StopCPUProfile()
}

// realProfiler is the production implementation that delegates to runtime/pprof.
// It lives in its own file so the public surface in collector.go stays compact.
type realProfiler struct{}

func (p *realProfiler) StartCPUProfile(w io.Writer) error {
    return pprof.StartCPUProfile(w)
}

func (p *realProfiler) StopCPUProfile() {
    pprof.StopCPUProfile()
}
