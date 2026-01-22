package store

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Idempotency interface {
	SetIfNotExists(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

// RedisIdem uses Redis SETNX for idempotency keys.
type RedisIdem struct {
	rdb *redis.Client
}

func NewRedisIdem(rdb *redis.Client) *RedisIdem {
	return &RedisIdem{rdb: rdb}
}

func (r *RedisIdem) SetIfNotExists(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return r.rdb.SetNX(ctx, key, "1", ttl).Result()
}

// MemoryIdem provides a process-local fallback.
type MemoryIdem struct {
	mu    sync.Mutex
	items map[string]time.Time
}

func NewMemoryIdem() *MemoryIdem {
	m := &MemoryIdem{items: make(map[string]time.Time)}
	go m.cleanupLoop()
	return m
}

func (m *MemoryIdem) SetIfNotExists(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if exp, ok := m.items[key]; ok && time.Now().Before(exp) {
		return false, nil
	}

	m.items[key] = time.Now().Add(ttl)
	return true, nil
}

func (m *MemoryIdem) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		m.mu.Lock()
		for k, exp := range m.items {
			if now.After(exp) {
				delete(m.items, k)
			}
		}
		m.mu.Unlock()
	}
}
