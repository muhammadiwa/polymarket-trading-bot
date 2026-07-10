package adapters

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type RedisRisk struct {
	client               *redis.Client
	logger               *zap.Logger
	marketPositionLimit  int64
	strategyPositionLimit int64
}

func NewRedisRisk(url string, logger *zap.Logger, marketLimit, strategyLimit int64) (*RedisRisk, error) {
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

	return &RedisRisk{
		client:               client,
		logger:               logger,
		marketPositionLimit:  marketLimit,
		strategyPositionLimit: strategyLimit,
	}, nil
}

func sanitizeRedisKeyComponent(id string) string {
	return strings.ReplaceAll(id, ":", "_")
}

func (r *RedisRisk) CheckRisk(ctx context.Context, marketID, strategyID string, orderSize decimal.Decimal) (*ports.RiskDecision, error) {
	checkCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()

	sanitizedMarket := sanitizeRedisKeyComponent(marketID)
	sanitizedStrategy := sanitizeRedisKeyComponent(strategyID)

	budgetKey := "pqap:risk:daily_budget_remaining"
	budgetStr, err := r.client.Get(checkCtx, budgetKey).Result()
	if err != nil && err != redis.Nil {
		r.logger.Error("failed to get daily budget", zap.Error(err))
		return &ports.RiskDecision{
			Allowed: false,
			Reason:  "risk_check_error: failed to get daily budget",
		}, nil
	}

	if err != redis.Nil {
		budget, parseErr := decimal.NewFromString(budgetStr)
		if parseErr == nil && budget.LessThanOrEqual(decimal.Zero) {
			return &ports.RiskDecision{
				Allowed: false,
				Reason:  "daily_budget_exhausted",
			}, nil
		}
	}

	marketKey := fmt.Sprintf("pqap:risk:market_position:%s", sanitizedMarket)
	marketPosStr, err := r.client.Get(checkCtx, marketKey).Result()
	if err != nil && err != redis.Nil {
		r.logger.Error("failed to get market position", zap.Error(err))
		return &ports.RiskDecision{
			Allowed: false,
			Reason:  "risk_check_error: failed to get market position",
		}, nil
	}

	if err != redis.Nil {
		marketPos, parseErr := decimal.NewFromString(marketPosStr)
		if parseErr == nil && marketPos.GreaterThan(decimal.NewFromInt(r.marketPositionLimit)) {
			return &ports.RiskDecision{
				Allowed: false,
				Reason:  "per_market_position_limit_exceeded",
			}, nil
		}
	}

	strategyKey := fmt.Sprintf("pqap:risk:strategy_position:%s", sanitizedStrategy)
	strategyPosStr, err := r.client.Get(checkCtx, strategyKey).Result()
	if err != nil && err != redis.Nil {
		r.logger.Error("failed to get strategy position", zap.Error(err))
		return &ports.RiskDecision{
			Allowed: false,
			Reason:  "risk_check_error: failed to get strategy position",
		}, nil
	}

	if err != redis.Nil {
		strategyPos, parseErr := decimal.NewFromString(strategyPosStr)
		if parseErr == nil && strategyPos.GreaterThan(decimal.NewFromInt(r.strategyPositionLimit)) {
			return &ports.RiskDecision{
				Allowed: false,
				Reason:  "per_strategy_position_limit_exceeded",
			}, nil
		}
	}

	return &ports.RiskDecision{
		Allowed: true,
		Reason:  "",
	}, nil
}

func (r *RedisRisk) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisRisk) Close() error {
	return r.client.Close()
}

// GetExecutionMode reads the execution mode from Redis.
func (r *RedisRisk) GetExecutionMode(ctx context.Context) (string, error) {
	val, err := r.client.Get(ctx, "pqap:execution_mode").Result()
	if err == redis.Nil {
		return "PAPER", nil // Default to PAPER (safe mode)
	}
	if err != nil {
		return "", err
	}
	return val, nil
}
