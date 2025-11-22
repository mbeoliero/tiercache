package cacher

import "context"

// Interface defines the core interface for cache storage
type Interface[K comparable, V any] interface {
	BaseInfo
	MGet(ctx context.Context, keys []K) (map[K]V, []K, error)
	MSet(ctx context.Context, entities map[K]V) error
	MDel(ctx context.Context, keys []K) error
}

type BaseInfo interface {
	Name() string
}

type RunInfo interface {
	// Level returns the current cache level index, starting from 1 (e.g. 1, 2, 3...).
	Level() int
}
