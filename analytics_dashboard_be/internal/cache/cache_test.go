package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func TestMemoryRateLimiter(t *testing.T) {
	rl := NewMemoryRateLimiter()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		allowed, _, _ := rl.Allow(ctx, "k", 3, 60)
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	allowed, retry, _ := rl.Allow(ctx, "k", 3, 60)
	if allowed {
		t.Fatal("4th request should be denied")
	}
	if retry <= 0 {
		t.Fatalf("retryAfter = %d, want > 0", retry)
	}
	// Different key has its own budget.
	if ok, _, _ := rl.Allow(ctx, "other", 3, 60); !ok {
		t.Fatal("separate key should be allowed")
	}
}

func TestNoopCache(t *testing.T) {
	var n NoopCache
	var dest string
	ok, err := n.Get(context.Background(), "k", &dest)
	if ok || err != nil {
		t.Fatal("noop Get should always miss with no error")
	}
	if err := n.Set(context.Background(), "k", "v", 60); err != nil {
		t.Fatal("noop Set should be a no-op")
	}
}

func newMiniRedis(t *testing.T) *RedisCache {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	c, err := NewRedisCache("redis://" + mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRedisCacheGetSet(t *testing.T) {
	c := newMiniRedis(t)
	ctx := context.Background()

	var miss int
	if ok, _ := c.Get(ctx, "x", &miss); ok {
		t.Fatal("expected miss on empty cache")
	}
	if err := c.Set(ctx, "x", 42, 60); err != nil {
		t.Fatal(err)
	}
	var got int
	ok, err := c.Get(ctx, "x", &got)
	if err != nil || !ok || got != 42 {
		t.Fatalf("get = %v ok=%v err=%v", got, ok, err)
	}
}

func TestRedisRateLimiter(t *testing.T) {
	c := newMiniRedis(t)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if ok, _, _ := c.Allow(ctx, "u", 2, 60); !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	ok, retry, err := c.Allow(ctx, "u", 2, 60)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("over-limit request should be denied")
	}
	if retry <= 0 {
		t.Fatalf("retryAfter = %d, want > 0", retry)
	}
}
