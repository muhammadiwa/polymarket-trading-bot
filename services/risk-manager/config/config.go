package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Config struct {
	DailyLossLimitPct            float64
	MarketLimitPct               float64
	StrategyLimitPct             float64
	StateRefreshInterval         time.Duration
	StateTTL                     time.Duration
	DailyBudgetWarningThreshold  float64
	ReconstructionTimeout        time.Duration
	CapitalTotal                 string // #21: parsed as decimal to avoid float64 precision loss
	NATSURL                      string
	RedisURL                     string
	PostgresURL                  string
	MetricsPort                  string
	MetricsBindAddress           string
	APIPort                      string
	APIBindAddress               string
	LogLevel                     string
	JWTSecret                    string
	DrawdownLimitPct             float64
	DrawdownWarningThreshold     float64
	APITimeoutMinutes            int
	ReconMismatchLimit           int
	OrderCancelTimeout           time.Duration
	EmergencyStopTTL             time.Duration
	MaxCorrelatedPositions       int
	BatasiWinThreshold           int
	BatasiWinCooldownMin         int
	MetabolicCPUPercent          float64
	MetabolicMemoryBytes         uint64
	MetabolicGoroutines          int
}

func Load() *Config {
	return &Config{
		DailyLossLimitPct:           envFloatOrDefault("RISK_DAILY_LOSS_LIMIT_PCT", 0.02),
		MarketLimitPct:              envFloatOrDefault("RISK_MARKET_LIMIT_PCT", 0.10),
		StrategyLimitPct:            envFloatOrDefault("RISK_STRATEGY_LIMIT_PCT", 0.20),
		StateRefreshInterval:        envDurationOrDefault("RISK_STATE_REFRESH_INTERVAL", 30*time.Second),
		StateTTL:                    envDurationOrDefault("RISK_STATE_TTL", 60*time.Second),
		DailyBudgetWarningThreshold: envFloatOrDefault("RISK_DAILY_BUDGET_WARNING_THRESHOLD", 0.80),
		ReconstructionTimeout:       envDurationOrDefault("RISK_RECONSTRUCTION_TIMEOUT", 30*time.Second),
		CapitalTotal:                envOrDefault("CAPITAL_TOTAL", "10000.00"),
		NATSURL:                     envOrDefault("NATS_URL", "nats://localhost:4222"),
		RedisURL:                    envOrDefault("REDIS_URL", "localhost:6379"),
		PostgresURL:                 envOrDefault("POSTGRES_URL", "postgres://localhost:5432/pqap"),
		MetricsPort:                 envOrDefault("RISK_METRICS_PORT", "9093"),
		MetricsBindAddress:          envOrDefault("RISK_METRICS_BIND", "0.0.0.0"),
		APIPort:                     envOrDefault("RISK_API_PORT", "8080"),
		APIBindAddress:              envOrDefault("RISK_API_BIND", "0.0.0.0"),
		LogLevel:                    envOrDefault("RISK_LOG_LEVEL", "info"),
		JWTSecret:                   envOrDefault("JWT_SECRET", ""),
		DrawdownLimitPct:            envFloatOrDefault("RISK_DRAWDOWN_LIMIT_PCT", 0.10),
		DrawdownWarningThreshold:    envFloatOrDefault("RISK_DRAWDOWN_WARNING_THRESHOLD", 0.80),
		APITimeoutMinutes:           envIntOrDefault("RISK_API_TIMEOUT_MINUTES", 5),
		ReconMismatchLimit:          envIntOrDefault("RISK_RECON_MISMATCH_LIMIT", 3),
		OrderCancelTimeout:          envDurationOrDefault("RISK_ORDER_CANCEL_TIMEOUT", 5*time.Second),
		EmergencyStopTTL:            envDurationOrDefault("RISK_EMERGENCY_STOP_TTL", 60*time.Second),
		MaxCorrelatedPositions:      envIntOrDefault("RISK_MAX_CORRELATED_POSITIONS", 3),
		BatasiWinThreshold:          envIntOrDefault("RISK_BATASI_WIN_THRESHOLD", 5),
		BatasiWinCooldownMin:        envIntOrDefault("RISK_BATASI_WIN_COOLDOWN_MIN", 0),
		MetabolicCPUPercent:         envFloatOrDefault("RISK_METABOLIC_CPU_PERCENT", 80),
		MetabolicMemoryBytes:        envUint64OrDefault("RISK_METABOLIC_MEMORY_BYTES", 1073741824),
		MetabolicGoroutines:         envIntOrDefault("RISK_METABOLIC_GOROUTINES", 10000),
	}
}

func (c *Config) Validate() error {
	if c.DailyLossLimitPct <= 0 || c.DailyLossLimitPct > 1 {
		return fmt.Errorf("daily loss limit pct must be between 0 and 1, got %f", c.DailyLossLimitPct)
	}
	if c.MarketLimitPct <= 0 || c.MarketLimitPct > 1 {
		return fmt.Errorf("market limit pct must be between 0 and 1, got %f", c.MarketLimitPct)
	}
	if c.StrategyLimitPct <= 0 || c.StrategyLimitPct > 1 {
		return fmt.Errorf("strategy limit pct must be between 0 and 1, got %f", c.StrategyLimitPct)
	}
	if c.StateRefreshInterval <= 0 {
		return fmt.Errorf("state refresh interval must be > 0, got %v", c.StateRefreshInterval)
	}
	if c.StateTTL <= 0 {
		return fmt.Errorf("state TTL must be > 0, got %v", c.StateTTL)
	}
	if c.StateTTL <= c.StateRefreshInterval {
		return fmt.Errorf("state TTL (%v) must be greater than refresh interval (%v)", c.StateTTL, c.StateRefreshInterval)
	}
	if c.DailyBudgetWarningThreshold <= 0 || c.DailyBudgetWarningThreshold > 1 {
		return fmt.Errorf("daily budget warning threshold must be between 0 and 1, got %f", c.DailyBudgetWarningThreshold)
	}
	if c.CapitalTotal == "" {
		return fmt.Errorf("capital total must not be empty")
	}
	capitalDec, err := decimal.NewFromString(c.CapitalTotal)
	if err != nil {
		return fmt.Errorf("capital total must be a valid decimal, got %q: %w", c.CapitalTotal, err)
	}
	if capitalDec.IsNegative() || capitalDec.IsZero() {
		return fmt.Errorf("capital total must be > 0, got %s", c.CapitalTotal)
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
	if c.DrawdownLimitPct <= 0 || c.DrawdownLimitPct > 1 {
		return fmt.Errorf("drawdown limit pct must be between 0 and 1, got %f", c.DrawdownLimitPct)
	}
	if c.DrawdownWarningThreshold <= 0 || c.DrawdownWarningThreshold > 1 {
		return fmt.Errorf("drawdown warning threshold must be between 0 and 1, got %f", c.DrawdownWarningThreshold)
	}
	if c.APITimeoutMinutes <= 0 {
		return fmt.Errorf("API timeout minutes must be > 0, got %d", c.APITimeoutMinutes)
	}
	if c.ReconMismatchLimit <= 0 {
		return fmt.Errorf("recon mismatch limit must be > 0, got %d", c.ReconMismatchLimit)
	}
	if c.OrderCancelTimeout <= 0 {
		return fmt.Errorf("order cancel timeout must be > 0, got %v", c.OrderCancelTimeout)
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET must not be empty")
	}
	if c.MaxCorrelatedPositions <= 0 {
		return fmt.Errorf("max correlated positions must be > 0, got %d", c.MaxCorrelatedPositions)
	}
	if c.BatasiWinThreshold <= 0 {
		return fmt.Errorf("batasi win threshold must be > 0, got %d", c.BatasiWinThreshold)
	}
	if c.MetabolicCPUPercent <= 0 || c.MetabolicCPUPercent > 100 {
		return fmt.Errorf("metabolic CPU percent must be between 0 and 100, got %f", c.MetabolicCPUPercent)
	}
	if c.MetabolicMemoryBytes <= 0 {
		return fmt.Errorf("metabolic memory bytes must be > 0, got %d", c.MetabolicMemoryBytes)
	}
	if c.MetabolicGoroutines <= 0 {
		return fmt.Errorf("metabolic goroutines must be > 0, got %d", c.MetabolicGoroutines)
	}
	return nil
}

func RedactCredentials(url string) string {
	idx := strings.LastIndex(url, "@")
	if idx == -1 {
		return url
	}
	schemeEnd := strings.Index(url, "://")
	if schemeEnd != -1 {
		return url[:schemeEnd+3] + "***:***" + url[idx:]
	}
	if idx > 0 {
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

func envIntOrDefault(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
		log.Printf("warning: invalid int value for %s=%q, using default %d", key, v, defaultVal)
	}
	return defaultVal
}

func envUint64OrDefault(key string, defaultVal uint64) uint64 {
	if v := os.Getenv(key); v != "" {
		u, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			return u
		}
		log.Printf("warning: invalid uint64 value for %s=%q, using default %d", key, v, defaultVal)
	}
	return defaultVal
}
