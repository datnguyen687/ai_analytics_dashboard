package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements domain.Cache over Redis, storing values as JSON.
type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(redisURL string) (*RedisCache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &RedisCache{client: client}, nil
}

func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	b, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, b, time.Duration(ttlSeconds)*time.Second).Err()
}

// DeleteByPrefix removes every key starting with prefix, scanning in batches so
// it never blocks Redis with a big KEYS call.
func (c *RedisCache) DeleteByPrefix(ctx context.Context, prefix string) error {
	var cursor uint64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, prefix+"*", 200).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		if next == 0 {
			return nil
		}
		cursor = next
	}
}

// Allow implements domain.RateLimiter with a fixed-window counter: INCR the key
// and set the TTL on first hit; over the limit → denied with the reset time.
// Redis errors fail OPEN so a cache outage never blocks the API.
func (c *RedisCache) Allow(ctx context.Context, key string, limit, windowSeconds int) (bool, int, error) {
	full := "rl:" + key
	count, err := c.client.Incr(ctx, full).Result()
	if err != nil {
		return true, 0, err
	}
	if count == 1 {
		_ = c.client.Expire(ctx, full, time.Duration(windowSeconds)*time.Second).Err()
	}
	if count > int64(limit) {
		ttl, err := c.client.TTL(ctx, full).Result()
		retry := windowSeconds
		if err == nil && ttl > 0 {
			retry = int(ttl.Seconds()) + 1
		}
		return false, retry, nil
	}
	return true, 0, nil
}

// NoopCache is used when Redis is unavailable — every Get misses, Set is a no-op,
// so the API degrades gracefully to always-fresh computation.
type NoopCache struct{}

func (NoopCache) Get(context.Context, string, interface{}) (bool, error) { return false, nil }
func (NoopCache) Set(context.Context, string, interface{}, int) error    { return nil }
func (NoopCache) DeleteByPrefix(context.Context, string) error           { return nil }
