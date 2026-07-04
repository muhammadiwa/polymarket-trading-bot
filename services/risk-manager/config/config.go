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
		JWTSecret:                   envOrDefault("RISK_JWT_SECRET", ""),
		DrawdownLimitPct:            envFloatOrDefault("RISK_DRAWDOWN_LIMIT_PCT", 0.10),
		DrawdownWarningThreshold:    envFloatOrDefault("RISK_DRAWDOWN_WARNING_THRESHOLD", 0.80),
		APITimeoutMinutes:           envIntOrDefault("RISK_API_TIMEOUT_MINUTES", 5),
		ReconMismatchLimit:          envIntOrDefault("RISK_RECON_MISMATCH_LIMIT", 3),
		OrderCancelTimeout:          envDurationOrDefault("RISK_ORDER_CANCEL_TIMEOUT", 5*time.Second),
		EmergencyStopTTL:            envDurationOrDefault("RISK_EMERGENCY_STOP_TTL", 60*time.Second),
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
	return "***:***" + url[idx:]
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
