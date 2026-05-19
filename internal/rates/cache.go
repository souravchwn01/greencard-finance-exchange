package rates

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *redis.Client
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

func cacheKey(pair string) string {
	return "rate:" + NormalizePair(pair)
}

func (c *Cache) SetLatest(ctx context.Context, rate CachedRate) error {
	rate.Pair = NormalizePair(rate.Pair)
	b, err := json.Marshal(rate)
	if err != nil {
		return fmt.Errorf("marshal cached rate: %w", err)
	}
	if err := c.rdb.Set(ctx, cacheKey(rate.Pair), b, 0).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

func (c *Cache) GetAllLatest(ctx context.Context) ([]CachedRate, error) {
	var cursor uint64
	var results []CachedRate
	for {
		keys, next, err := c.rdb.Scan(ctx, cursor, "rate:*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan: %w", err)
		}
		for _, key := range keys {
			val, err := c.rdb.Get(ctx, key).Bytes()
			if err != nil {
				if err == redis.Nil {
					continue
				}
				return nil, fmt.Errorf("redis get %s: %w", key, err)
			}
			var cr CachedRate
			if err := json.Unmarshal(val, &cr); err != nil {
				continue
			}
			cr.Pair = NormalizePair(cr.Pair)
			results = append(results, cr)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return results, nil
}

func (c *Cache) GetLatest(ctx context.Context, pair string) (*CachedRate, error) {
	pair = NormalizePair(pair)
	val, err := c.rdb.Get(ctx, cacheKey(pair)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrRateNotFound
		}
		return nil, fmt.Errorf("redis get: %w", err)
	}
	var out CachedRate
	if err := json.Unmarshal(val, &out); err != nil {
		return nil, fmt.Errorf("unmarshal cached rate: %w", err)
	}
	// Normalize in case old payloads existed.
	out.Pair = NormalizePair(out.Pair)
	if out.UpdatedAt.IsZero() {
		out.UpdatedAt = time.Now().UTC()
	}
	return &out, nil
}

