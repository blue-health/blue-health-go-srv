package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto"
)

type Memory struct {
	cache *ristretto.Cache
}

const (
	bufferItems = 64
)

func NewMemory(numCounters, maxCost int64) (*Memory, error) {
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: numCounters,
		MaxCost:     maxCost,
		BufferItems: bufferItems,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	return &Memory{cache: c}, nil
}

func (c *Memory) Get(_ context.Context, key string) ([]byte, error) {
	i, f := c.cache.Get(key)
	if !f {
		return nil, nil
	}

	v, ok := i.([]byte)
	if !ok {
		return nil, nil
	}

	return v, nil
}

func (c *Memory) Set(_ context.Context, key string, value []byte, expiry time.Duration) error {
	_ = c.cache.SetWithTTL(key, value, 1, expiry)
	return nil
}

func (c *Memory) Del(_ context.Context, key string) error {
	c.cache.Del(key)
	return nil
}
