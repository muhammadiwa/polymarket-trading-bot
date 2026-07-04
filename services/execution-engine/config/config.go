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
	SlippageTolerance        float64
	TimeInForce              string
	CircuitBreakerThreshold  int
	CircuitBreakerCooldown   time.Duration
	CircuitBreakerProbeTimeout time.Duration
	FillPollInterval         time.Duration
	FillPollTimeout          time.Duration
	MaxRetries               int
	RetryBackoffInitial      time.Duration
	RetryBackoffMax          time.Duration
	MaxConcurrency           int
	AtomicTimeout            time.Duration
	LegCancelTimeout         time.Duration
	NATSURL                  string
	RedisURL                 string
	PostgresURL              string
	PolymarketCLOBURL        string
	MetricsPort              string
	MetricsBindAddress       string
	LogLevel                 string
	JWTSecret                string
}

func Load() *Config {
	return &Config{
		SlippageTolerance:          envFloatOrDefault("EXECUTION_SLIPPAGE_TOLERANCE", 0.01),
		TimeInForce:                envOrDefault("EXECUTION_TIME_IN_FORCE", "GTC"),
		CircuitBreakerThreshold:    envIntOrDefault("EXECUTION_CIRCUIT_BREAKER_THRESHOLD", 5),
		CircuitBreakerCooldown:     envDurationOrDefault("EXECUTION_CIRCUIT_BREAKER_COOLDOWN", 60*time.Second),
		CircuitBreakerProbeTimeout: envDurationOrDefault("EXECUTION_CIRCUIT_BREAKER_PROBE_TIMEOUT", 5*time.Second),
		FillPollInterval:           envDurationOrDefault("EXECUTION_FILL_POLL_INTERVAL", 100*time.Millisecond),
		FillPollTimeout:            envDurationOrDefault("EXECUTION_FILL_POLL_TIMEOUT", 30*time.Second),
		MaxRetries:                 envIntOrDefault("EXECUTION_MAX_RETRIES", 3),
		RetryBackoffInitial:        envDurationOrDefault("EXECUTION_RETRY_BACKOFF_INITIAL", 100*time.Millisecond),
		RetryBackoffMax:            envDurationOrDefault("EXECUTION_RETRY_BACKOFF_MAX", 5*time.Second),
		MaxConcurrency:             envIntOrDefault("EXECUTION_MAX_CONCURRENCY", 10),
		AtomicTimeout:              envDurationOrDefault("EXECUTION_ATOMIC_TIMEOUT", 500*time.Millisecond),
		LegCancelTimeout:           envDurationOrDefault("EXECUTION_LEG_CANCEL_TIMEOUT", 1000*time.Millisecond),
		NATSURL:                    envOrDefault("NATS_URL", "nats://localhost:4222"),
		RedisURL:                   envOrDefault("REDIS_URL", "localhost:6379"),
		PostgresURL:                envOrDefault("POSTGRES_URL", "postgres://localhost:5432/pqap"),
		PolymarketCLOBURL:          envOrDefault("POLYMARKET_CLOB_URL", "https://clob.polymarket.com"),
		MetricsPort:                envOrDefault("EXECUTION_METRICS_PORT", "9092"),
		MetricsBindAddress:         envOrDefault("EXECUTION_METRICS_BIND", "0.0.0.0"),
		LogLevel:                   envOrDefault("EXECUTION_LOG_LEVEL", "info"),
		JWTSecret:                  envOrDefault("EXECUTION_JWT_SECRET", ""),
	}
}

func (c *Config) Validate() error {
	if c.SlippageTolerance < 0 || c.SlippageTolerance > 1 {
		return fmt.Errorf("slippage tolerance must be between 0 and 1, got %f", c.SlippageTolerance)
	}
	if c.CircuitBreakerThreshold < 1 {
		return fmt.Errorf("circuit breaker threshold must be >= 1, got %d", c.CircuitBreakerThreshold)
	}
	if c.CircuitBreakerCooldown <= 0 {
		return fmt.Errorf("circuit breaker cooldown must be > 0, got %v", c.CircuitBreakerCooldown)
	}
	if c.CircuitBreakerProbeTimeout <= 0 {
		return fmt.Errorf("circuit breaker probe timeout must be > 0, got %v", c.CircuitBreakerProbeTimeout)
	}
	if c.FillPollInterval <= 0 {
		return fmt.Errorf("fill poll interval must be > 0, got %v", c.FillPollInterval)
	}
	if c.FillPollTimeout <= 0 {
		return fmt.Errorf("fill poll timeout must be > 0, got %v", c.FillPollTimeout)
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries must be >= 0, got %d", c.MaxRetries)
	}
	if c.RetryBackoffInitial <= 0 {
		return fmt.Errorf("retry backoff initial must be > 0, got %v", c.RetryBackoffInitial)
	}
	if c.RetryBackoffMax <= 0 {
		return fmt.Errorf("retry backoff max must be > 0, got %v", c.RetryBackoffMax)
	}
	if c.MaxConcurrency < 1 {
		return fmt.Errorf("max concurrency must be >= 1, got %d", c.MaxConcurrency)
	}
	if c.AtomicTimeout <= 0 {
		return fmt.Errorf("atomic timeout must be > 0, got %v", c.AtomicTimeout)
	}
	if c.LegCancelTimeout <= 0 {
		return fmt.Errorf("leg cancel timeout must be > 0, got %v", c.LegCancelTimeout)
	}
	if c.NATSURL == "" {
		return fmt.Errorf("NATS_URL must not be empty")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL must not be empty")
	}
	if c.PostgresURL == "" {
		return fmt.Errorf("POSTGRES_URL must not be empty")
	}
	if c.PolymarketCLOBURL == "" {
		return fmt.Errorf("POLYMARKET_CLOB_URL must not be empty")
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
