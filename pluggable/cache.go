package pluggable

import (
	"context"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

type Cacheable interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	Get(ctx context.Context, key string) (interface{}, error)
}

type memoryCache struct {
	c *cache.Cache
}

func (m *memoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.c.Set(key, value, ttl)
	return nil
}

func (m *memoryCache) Get(ctx context.Context, key string) (interface{}, error) {
	data, ok := m.c.Get(key)
	if !ok {
		return nil, errors.Errorf("cache not found %s", key)
	}
	return data, nil
}
