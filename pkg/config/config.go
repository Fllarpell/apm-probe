package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the configuration for the APM probe.
// It's populated from environment variables.
type Config struct {
	Enabled                   bool
	DebugEndpoint             string
	CollectionInterval        time.Duration
	ProfilingEnabled          bool
	ProfilingLatencyThreshold time.Duration
	ProfilingDuration         time.Duration `json:"profiling_duration_s"`
	ProfilingCooldown         time.Duration `json:"profiling_cooldown_s"`
	ProfilingErrorThreshold   int           `json:"profiling_error_threshold"`
	NPlusOneThreshold         int           `json:"n_plus_one_threshold"`
}

// Load reads configuration from environment variables and returns a Config struct.
func Load() *Config {
	return &Config{
		Enabled:                   getEnvAsBool("APM_ENABLED", true),
		DebugEndpoint:             getEnv("APM_DEBUG_ENDPOINT", "/debug/apm"),
		CollectionInterval:        getEnvAsDuration("APM_COLLECTION_INTERVAL_S", 10*time.Second),
		ProfilingEnabled:          getEnvAsBool("APM_PROFILING_ENABLED", true),
		ProfilingLatencyThreshold: getEnvAsDurationMs("APM_PROFILING_LATENCY_THRESHOLD_MS", 500*time.Millisecond),
		ProfilingDuration:         getEnvAsDuration("APM_PROFILING_DURATION_S", 30*time.Second),
		ProfilingCooldown:         getEnvAsDuration("APM_PROFILING_COOLDOWN_S", 300*time.Second), // 5 minutes
		ProfilingErrorThreshold:   getEnvAsInt("APM_PROFILING_ERROR_THRESHOLD_COUNT", 5),
		NPlusOneThreshold:         getEnvAsInt("APM_N_PLUS_ONE_THRESHOLD_COUNT", 10),
	}
}

// getEnv reads an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsBool reads a boolean environment variable or returns a default value.
func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvAsDuration reads a duration environment variable (in seconds) or returns a default value.
// getEnvAsInt reads an integer environment variable or returns a default value.
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsDuration reads a duration environment variable (in seconds) or returns a default value.
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return time.Duration(intValue) * time.Second
		}
	}
	return defaultValue
}

// getEnvAsDurationMs reads a duration environment variable (in milliseconds) or returns a default value.
func getEnvAsDurationMs(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return time.Duration(intValue) * time.Millisecond
		}
	}
	return defaultValue
}
