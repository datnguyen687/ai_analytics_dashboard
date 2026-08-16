package cache

import (
	"context"
	"sync"
	"time"
)

// MemoryRateLimiter is the in-process fallback used when Redis is unavailable.
// Fixed-window counters guarded by a mutex — correct for a single instance
// (each instance would count independently if horizontally scaled).
type MemoryRateLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
}

type window struct {
	count   int
	resetAt time.Time
}

func NewMemoryRateLimiter() *MemoryRateLimiter {
	return &MemoryRateLimiter{windows: make(map[string]*window)}
}

func (m *MemoryRateLimiter) Allow(_ context.Context, key string, limit, windowSeconds int) (bool, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	w, ok := m.windows[key]
	if !ok || now.After(w.resetAt) {
		m.windows[key] = &window{count: 1, resetAt: now.Add(time.Duration(windowSeconds) * time.Second)}
		return true, 0, nil
	}
	if w.count >= limit {
		retry := int(time.Until(w.resetAt).Seconds()) + 1
		if retry < 1 {
			retry = 1
		}
		return false, retry, nil
	}
	w.count++
	return true, 0, nil
}
