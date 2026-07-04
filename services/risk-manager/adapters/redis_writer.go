package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type RedisWriter struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisWriter(url string, logger *zap.Logger) (*RedisWriter, error) {
	opt, err := redis.ParseURL("redis://" + url)
	if err != nil {
		opt = &redis.Options{
			Addr: url,
		}
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisWriter{
		client: client,
		logger: logger,
	}, nil
}

func (w *RedisWriter) WriteState(state ports.PitBossState, ttl time.Duration, emergencyTTL time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipe := w.client.Pipeline()

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	pipe.Set(ctx, "pqap:risk:state", stateJSON, ttl)
	pipe.Set(ctx, "pqap:risk:daily_budget", state.DailyBudgetRemaining.String(), ttl)
	pipe.Set(ctx, "pqap:risk:daily_loss", state.DailyLoss.String(), ttl)

	for marketID, entry := range state.MarketLimits {
		key := fmt.Sprintf("pqap:risk:market:%s", sanitizeRedisKeyComponent(marketID)) // #15
		pipe.Set(ctx, key, entry.Exposure.String(), ttl)
	}

	for strategyID, entry := range state.StrategyLimits {
		key := fmt.Sprintf("pqap:risk:strategy:%s", sanitizeRedisKeyComponent(strategyID)) // #15
		pipe.Set(ctx, key, entry.Exposure.String(), ttl)
	}

	// #17: Write emergency_stop last as sentinel for consistency
	emergencyVal := "false"
	if state.EmergencyStop {
		emergencyVal = "true"
	}
	// #20: Use EmergencyStopTTL for emergency keys
	pipe.Set(ctx, "pqap:risk:emergency_stop", emergencyVal, emergencyTTL)
	pipe.Set(ctx, "pqap:risk:emergency_reason", state.EmergencyStopReason, emergencyTTL)
	if state.EmergencyStopTimestamp != nil {
		pipe.Set(ctx, "pqap:risk:emergency_timestamp", state.EmergencyStopTimestamp.Format(time.RFC3339), emergencyTTL)
	}
	pipe.Set(ctx, "pqap:risk:peak_equity", state.PeakEquity.String(), ttl)
	pipe.Set(ctx, "pqap:risk:current_equity", state.CurrentEquity.String(), ttl)
	pipe.Set(ctx, "pqap:risk:drawdown", state.Drawdown.String(), ttl)
	pipe.Set(ctx, "pqap:risk:drawdown_limit", state.DrawdownLimit.String(), ttl)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to write state to Redis: %w", err)
	}

	return nil
}

func (w *RedisWriter) ReadState() (*ports.PitBossState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	stateStr, err := w.client.Get(ctx, "pqap:risk:state").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("risk state not found in Redis (TTL expired)")
		}
		return nil, fmt.Errorf("failed to read state from Redis: %w", err)
	}

	var state ports.PitBossState
	if err := json.Unmarshal([]byte(stateStr), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

func (w *RedisWriter) ReadDailyBudget() (decimal.Decimal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	val, err := w.client.Get(ctx, "pqap:risk:daily_budget").Result()
	if err != nil {
		if err == redis.Nil {
			return decimal.Zero, nil
		}
		return decimal.Zero, fmt.Errorf("failed to read daily budget: %w", err)
	}

	return decimal.NewFromString(val)
}

func (w *RedisWriter) ReadEmergencyStop() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	val, err := w.client.Get(ctx, "pqap:risk:emergency_stop").Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("failed to read emergency stop: %w", err)
	}

	return val == "true", nil
}

func (w *RedisWriter) Close() error {
	return w.client.Close()
}

// sanitizeRedisKeyComponent strips characters that could cause key injection. #15
func sanitizeRedisKeyComponent(id string) string {
	sanitized := strings.ReplaceAll(id, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, "\n", "_")
	sanitized = strings.ReplaceAll(sanitized, "\r", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	return sanitized
}
