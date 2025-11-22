package cacher

// Middleware defines the middleware type
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
