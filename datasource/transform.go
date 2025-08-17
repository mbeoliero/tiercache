package datasource

import "context"

type Fetcher[K comparable, V any] func(ctx context.Context, key K) (V, error)

type BatchFetcher[K comparable, V any] func(ctx context.Context, keys []K) (map[K]V, error)

func (f Fetcher[K, V]) ToBatchFetcher() BatchFetcher[K, V] {
	return func(ctx context.Context, keys []K) (map[K]V, error) {
		ret := make(map[K]V)
		for _, k := range keys {
			v, err := f(ctx, k)
			if err != nil {
				return nil, err
			}
			ret[k] = v
		}
		return ret, nil
	}
}
