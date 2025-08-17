package tiercache

import "context"

type CacheStore[K comparable, V any] interface {
	MGet(ctx context.Context, keys []K) (map[K]V, []K, error)
	MSet(ctx context.Context, entities map[K]V) error
	MDel(ctx context.Context, keys []K) error
}
