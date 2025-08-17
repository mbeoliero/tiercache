package localcache

import (
	"context"
	"time"

	"github.com/maypok86/otter/v2"
)

type LocalCache[K comparable, V any] struct {
	cache *otter.Cache[K, V]
	ttl   time.Duration
}

func NewLocalCache[K comparable, V any](ttl time.Duration) *LocalCache[K, V] {
	return &LocalCache[K, V]{
		cache: otter.Must(&otter.Options[K, V]{
			MaximumSize:       10_000,
			ExpiryCalculator:  otter.ExpiryAccessing[K, V](ttl),
			RefreshCalculator: otter.RefreshWriting[K, V](1 * time.Second),
		}),
		ttl: ttl,
	}
}

func (r *LocalCache[K, V]) MGet(ctx context.Context, keys []K) (map[K]V, []K, error) {
	ret := make(map[K]V, len(keys))
	miss := make([]K, 0)
	if len(keys) == 0 {
		return ret, miss, nil
	}

	for _, key := range keys {
		if val, ok := r.cache.GetIfPresent(key); ok {
			ret[key] = val
		} else {
			miss = append(miss, key)
		}
	}

	return ret, miss, nil
}

func (r *LocalCache[K, V]) MSet(ctx context.Context, entities map[K]V) error {
	if len(entities) == 0 {
		return nil
	}

	for k, v := range entities {
		r.cache.Set(k, v)
	}

	return nil
}

func (r *LocalCache[K, T]) MDel(ctx context.Context, keys []K) error {
	if len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		r.cache.SetExpiresAfter(key, 1)
	}
	return nil
}
