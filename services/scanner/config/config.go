package config

import (
	"os"
	"time"
)

type Config struct {
	WSURL               string
	RESTURL             string
	StaleThreshold      time.Duration
	MarketPollInterval  time.Duration
	ReconnectInitial    time.Duration
	ReconnectMax        time.Duration
	NATSURL             string
	RedisURL            string
	MetricsPort         string
	MetricsBindAddress  string
	LogLevel            string
	MaxBatchSize        int
	RedisTTL            time.Duration
}

func Load() *Config {
	return &Config{
		WSURL:              envOrDefault("SCANNER_WS_URL", "wss://ws-subscriptions-clob.polymarket.com/ws/market"),
		RESTURL:            envOrDefault("SCANNER_REST_URL", "https://clob.polymarket.com"),
		StaleThreshold:     envDurationOrDefault("SCANNER_STALE_THRESHOLD", 30*time.Second),
		MarketPollInterval: envDurationOrDefault("SCANNER_MARKET_POLL_INTERVAL", 60*time.Second),
		ReconnectInitial:   envDurationOrDefault("SCANNER_RECONNECT_INITIAL", 1*time.Second),
		ReconnectMax:       envDurationOrDefault("SCANNER_RECONNECT_MAX", 60*time.Second),
		NATSURL:            envOrDefault("NATS_URL", "nats://localhost:4222"),
		RedisURL:           envOrDefault("REDIS_URL", "localhost:6379"),
		MetricsPort:        envOrDefault("SCANNER_METRICS_PORT", "9090"),
		MetricsBindAddress: envOrDefault("SCANNER_METRICS_BIND", "0.0.0.0"),
		LogLevel:           envOrDefault("SCANNER_LOG_LEVEL", "info"),
		MaxBatchSize:       100,
		RedisTTL:           5 * time.Second,
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil && d > 0 {
			return d
		}
	}
	return defaultVal
}
