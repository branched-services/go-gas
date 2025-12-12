// Package config provides environment-based configuration following 12-factor principles.
// All configuration is loaded from environment variables with the GAS_ prefix.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Config holds all service configuration.
// All fields are loaded from environment variables with the GAS_ prefix.
type Config struct {
	// Node connection (Factor IV: Backing Services)
	NodeWSURL   string
	NodeHTTPURL string

	// Server addresses
	GRPCAddr string
	HTTPAddr string

	// Estimator tuning
	HistoryBlocks  int
	MempoolSamples int
	RecalcInterval time.Duration

	// Observability
	LogLevel  string
	LogFormat string
}

// Load reads configuration from environment variables.
// All variables are prefixed with GAS_ (e.g., GAS_NODE_WS_URL).
func Load() (*Config, error) {
	cfg := &Config{
		// Required fields have no defaults
		NodeWSURL:   os.Getenv("GAS_NODE_WS_URL"),
		NodeHTTPURL: os.Getenv("GAS_NODE_HTTP_URL"),

		// Optional fields with defaults
		GRPCAddr:       envOrDefault("GAS_GRPC_ADDR", ":9090"),
		HTTPAddr:       envOrDefault("GAS_HTTP_ADDR", ":8080"),
		HistoryBlocks:  envIntOrDefault("GAS_HISTORY_BLOCKS", 20),
		MempoolSamples: envIntOrDefault("GAS_MEMPOOL_SAMPLES", 500),
		RecalcInterval: envDurationOrDefault("GAS_RECALC_INTERVAL", 200*time.Millisecond),
		LogLevel:       envOrDefault("GAS_LOG_LEVEL", "info"),
		LogFormat:      envOrDefault("GAS_LOG_FORMAT", "json"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.NodeWSURL == "" {
		return errors.New("GAS_NODE_WS_URL is required")
	}
	if _, err := url.Parse(c.NodeWSURL); err != nil {
		return fmt.Errorf("invalid GAS_NODE_WS_URL: %w", err)
	}

	if c.NodeHTTPURL == "" {
		return errors.New("GAS_NODE_HTTP_URL is required")
	}
	if _, err := url.Parse(c.NodeHTTPURL); err != nil {
		return fmt.Errorf("invalid GAS_NODE_HTTP_URL: %w", err)
	}

	if c.HistoryBlocks < 1 || c.HistoryBlocks > 1000 {
		return errors.New("GAS_HISTORY_BLOCKS must be between 1 and 1000")
	}

	if c.MempoolSamples < 0 || c.MempoolSamples > 10000 {
		return errors.New("GAS_MEMPOOL_SAMPLES must be between 0 and 10000")
	}

	if c.RecalcInterval < 10*time.Millisecond {
		return errors.New("GAS_RECALC_INTERVAL must be at least 10ms")
	}

	return nil
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func envIntOrDefault(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func envDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
