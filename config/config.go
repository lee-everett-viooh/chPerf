package config

import (
	"os"
	"strconv"
)

// Config holds application configuration.
type Config struct {
	ClickHouseDSN string
	SQLitePath    string
	DefaultConcurrency int
	MaxConcurrency    int
	DefaultIterations int
	DefaultDurationSec int
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		ClickHouseDSN:     getEnv("CLICKHOUSE_DSN", "clickhouse://127.0.0.1:9000/default"),
		SQLitePath:        getEnv("SQLITE_PATH", "./chperf.db"),
		DefaultConcurrency: getEnvInt("DEFAULT_CONCURRENCY", 5),
		MaxConcurrency:     getEnvInt("MAX_CONCURRENCY", 100),
		DefaultIterations:  getEnvInt("DEFAULT_ITERATIONS", 100),
		DefaultDurationSec: getEnvInt("DEFAULT_DURATION_SEC", 60),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
