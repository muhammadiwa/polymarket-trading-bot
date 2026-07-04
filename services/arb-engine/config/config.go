package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	MinProfitThreshold       string
	ScoreThreshold           string
	FillProbWeightOrderbook  float64
	FillProbWeightHistorical float64
	LiquidityMaxDepth        float64
	FillProbRequiredDepth    float64
	NATSURL                  string
	TimescaleURL             string
	MetricsPort              string
	MetricsBindAddress       string
	LogLevel                 string
}

func Load() *Config {
	return &Config{
		MinProfitThreshold:       envOrDefault("ARB_MIN_PROFIT_THRESHOLD", "0.01"),
		ScoreThreshold:           envOrDefault("ARB_SCORE_THRESHOLD", "0.01"),
		FillProbWeightOrderbook:  envFloatOrDefault("ARB_FILL_PROB_WEIGHT_ORDERBOOK", 0.7),
		FillProbWeightHistorical: envFloatOrDefault("ARB_FILL_PROB_WEIGHT_HISTORICAL", 0.3),
		LiquidityMaxDepth:        envFloatOrDefault("ARB_LIQUIDITY_MAX_DEPTH", 10000.0),
		FillProbRequiredDepth:    envFloatOrDefault("ARB_FILL_PROB_REQUIRED_DEPTH", 1000.0),
		NATSURL:                  envOrDefault("NATS_URL", "nats://localhost:4222"),
		TimescaleURL:             envOrDefault("TIMESCALE_URL", "postgres://localhost:5432/pqap"),
		MetricsPort:              envOrDefault("ARB_METRICS_PORT", "9091"),
		MetricsBindAddress:       envOrDefault("ARB_METRICS_BIND", "0.0.0.0"),
		LogLevel:                 envOrDefault("ARB_LOG_LEVEL", "info"),
	}
}

func RedactCredentials(url string) string {
	if idx := strings.Index(url, "@"); idx != -1 {
		schemeEnd := strings.Index(url, "://")
		if schemeEnd != -1 {
			return url[:schemeEnd+3] + "***:***" + url[idx:]
		}
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
