package tiercache

import (
	"context"
	"fmt"
	"time"
)

type Middleware[K comparable, V any] func(next CacheStore[K, V]) CacheStore[K, V]

type loggerWrapper[K comparable, V any] struct {
	next CacheStore[K, V]
}

// LoggerMiddleware 是一个工厂函数，用于创建日志中间件
func LoggerMiddleware[K comparable, V any]() Middleware[K, V] {
	return func(next CacheStore[K, V]) CacheStore[K, V] {
		return &loggerWrapper[K, V]{
			next: next,
		}
	}
}

func (l *loggerWrapper[K, V]) MGet(ctx context.Context, keys []K) (map[K]V, []K, error) {
	startTime := time.Now()

	fmt.Printf("[Logger] -> MGet: requesting %v keys\n", keys)
	found, missing, err := l.next.MGet(ctx, keys)

	duration := time.Since(startTime)
	if err != nil {
		fmt.Printf("[Logger] <- MGet: finished in %s with error: %v\n", duration, err)
	} else {
		fmt.Printf(
			"[Logger] <- MGet: finished in %s. Found: %v, Missing: %v\n",
			duration,
			found,
			missing,
		)
	}

	return found, missing, err
}

func (l *loggerWrapper[K, V]) MSet(ctx context.Context, items map[K]V) error {
	startTime := time.Now()
	fmt.Printf("[Logger] -> MSet: setting %v items\n", items)

	err := l.next.MSet(ctx, items)

	duration := time.Since(startTime)
	if err != nil {
		fmt.Printf("[Logger] <- MSet: finished in %s with error: %v\n", duration, err)
	} else {
		fmt.Printf("[Logger] <- MSet: finished in %s successfully\n", duration)
	}

	return err
}

func (l *loggerWrapper[K, V]) MDel(ctx context.Context, keys []K) error {
	startTime := time.Now()
	fmt.Printf("[Logger] -> Delete: deleting %v keys\n", keys)

	err := l.next.MDel(ctx, keys)

	duration := time.Since(startTime)
	if err != nil {
		fmt.Printf("[Logger] <- Delete: finished in %s with error: %v\n", duration, err)
	} else {
		fmt.Printf("[Logger] <- Delete: finished in %s successfully\n", duration)
	}

	return err
}
