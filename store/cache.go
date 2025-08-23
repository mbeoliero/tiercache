package store

import "context"

// Interface 定义缓存存储的核心接口
type Interface[K comparable, V any] interface {
	MGet(ctx context.Context, keys []K) (map[K]V, []K, error)
	MSet(ctx context.Context, entities map[K]V) error
	MDel(ctx context.Context, keys []K) error
}

// Middleware 定义中间件类型
type Middleware[K comparable, V any] func(next Interface[K, V]) Interface[K, V]

func ChainMw[K comparable, V any](mws ...Middleware[K, V]) Middleware[K, V] {
	return func(next Interface[K, V]) Interface[K, V] {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

func WrapperStore[K comparable, V any](store Interface[K, V], mws ...Middleware[K, V]) Interface[K, V] {
	if len(mws) == 0 {
		return store
	}

	mw := ChainMw(mws...)
	return mw(store)
}
