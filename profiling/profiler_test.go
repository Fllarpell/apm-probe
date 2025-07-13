package profiling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfiler_ProfileEndpointIfSlow(t *testing.T) {
	cfg := Config{
		Enabled:          true,
		LatencyThreshold: 100 * time.Millisecond,
		Duration:         1 * time.Second,
		Cooldown:         200 * time.Millisecond,
	}

	t.Run("should not trigger on fast endpoint", func(t *testing.T) {
		profiler := NewProfiler(cfg)
		require.NotNil(t, profiler)

		fastDuration := 50 * time.Millisecond
		profiler.ProfileEndpointIfSlow("/fast", fastDuration)

		assert.False(t, profiler.isCoolingDown("/fast"), "cooldown should not be set for a fast endpoint")
	})

	t.Run("should trigger on slow endpoint", func(t *testing.T) {
		profiler := NewProfiler(cfg)
		require.NotNil(t, profiler)

		slowDuration := 150 * time.Millisecond
		profiler.ProfileEndpointIfSlow("/slow", slowDuration)

		assert.True(t, profiler.isCoolingDown("/slow"), "cooldown should be set for a slow endpoint")
	})

	t.Run("should respect cooldown period", func(t *testing.T) {
		profiler := NewProfiler(cfg)
		require.NotNil(t, profiler)

		slowDuration := 150 * time.Millisecond
		profiler.ProfileEndpointIfSlow("/slow-cooldown", slowDuration)
		require.True(t, profiler.isCoolingDown("/slow-cooldown"), "cooldown should be set after the first slow request")

		cooldownEnd := profiler.cooldowns["/slow-cooldown"]

		profiler.ProfileEndpointIfSlow("/slow-cooldown", slowDuration)
		assert.Equal(t, cooldownEnd, profiler.cooldowns["/slow-cooldown"], "cooldown time should not be extended on second call")
	})

	t.Run("should allow profiling again after cooldown", func(t *testing.T) {
		profiler := NewProfiler(cfg)
		require.NotNil(t, profiler)

		profiler.ProfileEndpointIfSlow("/slow-after-cooldown", 150*time.Millisecond)
		require.True(t, profiler.isCoolingDown("/slow-after-cooldown"))

		time.Sleep(cfg.Cooldown + 50*time.Millisecond)

		assert.False(t, profiler.isCoolingDown("/slow-after-cooldown"), "cooldown should have expired")

		profiler.ProfileEndpointIfSlow("/slow-after-cooldown", 150*time.Millisecond)
		assert.True(t, profiler.isCoolingDown("/slow-after-cooldown"), "cooldown should be set again after it expires")
	})
}
