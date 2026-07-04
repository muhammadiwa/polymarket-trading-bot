package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	logger *zap.Logger
}

func NewRedisCache(url string, ttl time.Duration, logger *zap.Logger) (*RedisCache, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		opts = &redis.Options{
			Addr: url,
		}
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		ttl:    ttl,
		logger: logger,
	}, nil
}

func (rc *RedisCache) SetMarket(ctx context.Context, market catalog.Market) error {
	data, err := json.Marshal(market)
	if err != nil {
		return fmt.Errorf("failed to marshal market: %w", err)
	}

	key := fmt.Sprintf("pqap:market:%s", market.ID)
	if err := rc.client.Set(ctx, key, data, rc.ttl).Err(); err != nil {
		rc.logger.Error("failed to set market in redis", zap.Error(err), zap.String("key", key))
		return err
	}

	rc.client.SAdd(ctx, "pqap:markets:active", market.ID)

	return nil
}

func (rc *RedisCache) GetMarket(ctx context.Context, marketID string) (*catalog.Market, error) {
	key := fmt.Sprintf("pqap:market:%s", marketID)
	data, err := rc.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var market catalog.Market
	if err := json.Unmarshal(data, &market); err != nil {
		return nil, err
	}

	return &market, nil
}

func (rc *RedisCache) RemoveMarket(ctx context.Context, marketID string) error {
	key := fmt.Sprintf("pqap:market:%s", marketID)
	if err := rc.client.Del(ctx, key).Err(); err != nil {
		rc.logger.Error("failed to delete market from redis", zap.Error(err), zap.String("key", key))
		return err
	}
	if err := rc.client.SRem(ctx, "pqap:markets:active", marketID).Err(); err != nil {
		rc.logger.Error("failed to remove market from active set", zap.Error(err), zap.String("market_id", marketID))
		return err
	}
	return nil
}

func (rc *RedisCache) GetActiveMarketIDs(ctx context.Context) ([]string, error) {
	return rc.client.SMembers(ctx, "pqap:markets:active").Result()
}

func (rc *RedisCache) IsConnected() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return rc.client.Ping(ctx).Err() == nil
}

func (rc *RedisCache) Close() error {
	return rc.client.Close()
}
