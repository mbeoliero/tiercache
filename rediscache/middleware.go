package rediscache

import (
	"context"
	"fmt"
	"time"

	"github.com/mbeoliero/tiercache/store"
)

// metricsWrapper 为 Redis 缓存添加指标中间件
type metricsWrapper[K comparable, V any] struct {
	name string
	next store.Interface[K, V]
}

// MetricsMiddleware 创建指标收集中间件
func MetricsMiddleware[K comparable, V any](name string) store.Middleware[K, V] {
	return func(next store.Interface[K, V]) store.Interface[K, V] {
		return &metricsWrapper[K, V]{
			name: name,
			next: next,
		}
	}
}

func (m *metricsWrapper[K, V]) MGet(ctx context.Context, keys []K) (ret map[K]V, miss []K, err error) {
	start := time.Now()
	defer func() {
		// 这里可以添加指标收集逻辑
		fmt.Printf("[metricsWrapper] MGet took %v, input length: %d, miss length: %d, err: %v\n", time.Since(start), len(keys), len(miss), err)
	}()
	return m.next.MGet(ctx, keys)
}

func (m *metricsWrapper[K, V]) MSet(ctx context.Context, entities map[K]V) error {
	start := time.Now()
	defer func() {
		// 这里可以添加指标收集逻辑
		fmt.Printf("[metricsWrapper] MSet took %v, set length: %d\n", time.Since(start), len(entities))
	}()
	return m.next.MSet(ctx, entities)
}

func (m *metricsWrapper[K, V]) MDel(ctx context.Context, keys []K) error {
	start := time.Now()
	defer func() {
		// 这里可以添加指标收集逻辑
		fmt.Printf("[metricsWrapper] MDel took %v, del length: %d\n", time.Since(start), len(keys))
	}()
	return m.next.MDel(ctx, keys)
}
