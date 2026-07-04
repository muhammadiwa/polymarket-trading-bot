package adapters

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type RedisMarketPrice struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisMarketPrice(url string, logger *zap.Logger) (*RedisMarketPrice, error) {
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
		return nil, fmt.Errorf("failed to connect to Redis for market price: %w", err)
	}

	return &RedisMarketPrice{
		client: client,
		logger: logger,
	}, nil
}

func (r *RedisMarketPrice) GetCurrentPrice(ctx context.Context, marketID string) (decimal.Decimal, error) {
	sanitized := sanitizeRedisKeyComponent(marketID)
	key := fmt.Sprintf("pqap:market:price:%s", sanitized)

	checkCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	priceStr, err := r.client.Get(checkCtx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return decimal.Zero, fmt.Errorf("no current price found for market %s", marketID)
		}
		return decimal.Zero, fmt.Errorf("failed to get current price for market %s: %w", marketID, err)
	}

	price, err := decimal.NewFromString(strings.TrimSpace(priceStr))
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid price format for market %s: %w", marketID, err)
	}

	return price, nil
}

func (r *RedisMarketPrice) Close() error {
	return r.client.Close()
}
