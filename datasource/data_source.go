package datasource

import "context"

type DataSource[K comparable, V any] struct {
	batchFetch func(ctx context.Context, keys []K) (map[K]V, error)
}

func NewDataSource[K comparable, V any](f func(ctx context.Context, keys []K) (map[K]V, error)) *DataSource[K, V] {
	return &DataSource[K, V]{
		batchFetch: f,
	}
}

func NewDataSourceWithFetcher[K comparable, V any](f Fetcher[K, V]) *DataSource[K, V] {
	return &DataSource[K, V]{
		batchFetch: f.ToBatchFetcher(),
	}
}

func (r *DataSource[K, V]) MGet(ctx context.Context, keys []K) (map[K]V, []K, error) {
	ret, err := r.batchFetch(ctx, keys)
	if err != nil {
		return nil, nil, err
	}

	miss := make([]K, 0)
	for _, k := range keys {
		if _, ok := ret[k]; !ok {
			miss = append(miss, k)
		}
	}

	return ret, miss, nil
}

func (r *DataSource[K, V]) MSet(ctx context.Context, entities map[K]V) error {
	return nil
}

func (r *DataSource[K, T]) MDel(ctx context.Context, keys []K) error {
	return nil
}
