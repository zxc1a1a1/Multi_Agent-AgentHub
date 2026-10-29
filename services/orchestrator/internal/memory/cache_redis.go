package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
)

// RedisCache stores already-authorized recall results. Keys must be versioned
// by scope epoch; a stale key is never reused after a mutation.
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(client *redis.Client) (*RedisCache, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	return &RedisCache{client: client, ttl: 5 * time.Minute}, nil
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]rtmemory.Entry, bool) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var entries []rtmemory.Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, false
	}
	return entries, true
}

func (c *RedisCache) Set(ctx context.Context, key string, entries []rtmemory.Entry, ttl time.Duration) {
	if ttl <= 0 || ttl > c.ttl {
		ttl = c.ttl
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return
	}
	_ = c.client.Set(ctx, key, data, ttl).Err()
}
