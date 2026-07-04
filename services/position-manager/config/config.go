package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ReconciliationInterval       time.Duration
	ReconciliationMismatchThreshold int
	MarketLimitPct               float64
	StrategyLimitPct             float64
	TotalCapital                 float64
	PnLUpdateTimeoutMS           int
	ExitOrderTimeoutMS           int
	AlertDeliveryTimeoutMS       int
	NATSURL                      string
	PostgresURL                  string
	PolymarketAPIURL             string
	MetricsPort                  string
	MetricsBindAddress           string
	LogLevel                     string
	JWTSecret                    string
}

func Load() *Config {
	return &Config{
		ReconciliationInterval:          envDurationOrDefault("POSITION_RECONCILIATION_INTERVAL", 60*time.Second),
		ReconciliationMismatchThreshold: envIntOrDefault("POSITION_RECONCILIATION_MISMATCH_THRESHOLD", 3),
		MarketLimitPct:                  envFloatOrDefault("POSITION_MARKET_LIMIT_PCT", 0.10),
		StrategyLimitPct:                envFloatOrDefault("POSITION_STRATEGY_LIMIT_PCT", 0.20),
		TotalCapital:                    envFloatOrDefault("POSITION_TOTAL_CAPITAL", 10000.0),
		PnLUpdateTimeoutMS:              envIntOrDefault("POSITION_PNL_UPDATE_TIMEOUT_MS", 1000),
		ExitOrderTimeoutMS:              envIntOrDefault("POSITION_EXIT_ORDER_TIMEOUT_MS", 1000),
		AlertDeliveryTimeoutMS:          envIntOrDefault("POSITION_ALERT_DELIVERY_TIMEOUT_MS", 5000),
		NATSURL:                         envOrDefault("NATS_URL", "nats://localhost:4222"),
		PostgresURL:                     envOrDefault("POSTGRES_URL", "postgres://localhost:5432/pqap"),
		PolymarketAPIURL:                envOrDefault("POLYMARKET_API_URL", "https://gamma-api.polymarket.com"),
		MetricsPort:                     envOrDefault("POSITION_METRICS_PORT", "9093"),
		MetricsBindAddress:              envOrDefault("POSITION_METRICS_BIND", "0.0.0.0"),
		LogLevel:                        envOrDefault("POSITION_LOG_LEVEL", "info"),
		JWTSecret:                       envOrDefault("POSITION_JWT_SECRET", ""),
	}
}

func (c *Config) Validate() error {
	if c.ReconciliationInterval <= 0 {
		return fmt.Errorf("reconciliation interval must be > 0, got %v", c.ReconciliationInterval)
	}
	if c.ReconciliationMismatchThreshold < 1 {
		return fmt.Errorf("reconciliation mismatch threshold must be >= 1, got %d", c.ReconciliationMismatchThreshold)
	}
	if c.MarketLimitPct <= 0 || c.MarketLimitPct > 1 {
		return fmt.Errorf("market limit pct must be between 0 and 1, got %f", c.MarketLimitPct)
	}
	if c.StrategyLimitPct <= 0 || c.StrategyLimitPct > 1 {
		return fmt.Errorf("strategy limit pct must be between 0 and 1, got %f", c.StrategyLimitPct)
	}
	if c.NATSURL == "" {
		return fmt.Errorf("NATS_URL must not be empty")
	}
	if c.PostgresURL == "" {
		return fmt.Errorf("POSTGRES_URL must not be empty")
	}
	if c.PolymarketAPIURL == "" {
		return fmt.Errorf("POLYMARKET_API_URL must not be empty")
	}
	return nil
}

func RedactCredentials(url string) string {
	if idx := strings.Index(url, "@"); idx != -1 {
		schemeEnd := strings.Index(url, "://")
		if schemeEnd != -1 {
			return url[:schemeEnd+3] + "***:***" + url[idx:]
		}
		return "***:***" + url[idx:]
	}
	return url
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envFloatOrDefault(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f
		}
		log.Printf("warning: invalid float value for %s=%q, using default %v", key, v, defaultVal)
	}
	return defaultVal
}

func envIntOrDefault(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
		log.Printf("warning: invalid int value for %s=%q, using default %v", key, v, defaultVal)
	}
	return defaultVal
}

func envDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
		log.Printf("warning: invalid duration value for %s=%q, using default %v", key, v, defaultVal)
	}
	return defaultVal
}
