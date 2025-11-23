package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/mbeoliero/tiercache/cacher"
)

type loggerWrapper[K comparable, V any] struct {
	next cacher.Interface[K, V]
}

// LoggerMiddleware 是一个工厂函数，用于创建日志中间件
func LoggerMiddleware[K comparable, V any]() cacher.Middleware[K, V] {
	return func(next cacher.Interface[K, V]) cacher.Interface[K, V] {
		return &loggerWrapper[K, V]{
			next: next,
		}
	}
}

func (l *loggerWrapper[K, V]) MGet(ctx context.Context, keys []K) (map[K]V, []K, error) {
	startTime := time.Now()

	fmt.Printf("[Logger] [Level: %d, Name: %s] -> MGet: requesting %v keys\n", cacher.GetRunInfo(ctx).Level(), l.next.Name(), keys)
	found, missing, err := l.next.MGet(ctx, keys)

	duration := time.Since(startTime)
	if err != nil {
		fmt.Printf("[Logger] [Level: %d, Name: %s] <- MGet: finished in %s with error: %v\n", cacher.GetRunInfo(ctx).Level(), l.next.Name(), duration, err)
	} else {
		fmt.Printf(
			"[Logger] [Level: %d, Name: %s] <- MGet: finished in %s. Found: %v, Missing: %v\n",
			cacher.GetRunInfo(ctx).Level(),
			l.next.Name(),
			duration,
			found,
			missing,
		)
	}

	return found, missing, err
}

func (l *loggerWrapper[K, V]) MSet(ctx context.Context, items map[K]V) error {
	startTime := time.Now()
	fmt.Printf("[Logger] [Level: %d, Name: %s] -> MSet: setting %v items\n", cacher.GetRunInfo(ctx).Level(), l.next.Name(), items)

	err := l.next.MSet(ctx, items)

	duration := time.Since(startTime)
	if err != nil {
		fmt.Printf("[Logger] [Level: %d, Name: %s] <- MSet: finished in %s with error: %v\n", cacher.GetRunInfo(ctx).Level(), l.next.Name(), duration, err)
	} else {
		fmt.Printf("[Logger] [Level: %d, Name: %s] <- MSet: finished in %s successfully\n", cacher.GetRunInfo(ctx).Level(), l.next.Name(), duration)
	}

	return err
}

func (l *loggerWrapper[K, V]) MDel(ctx context.Context, keys []K) error {
	startTime := time.Now()
	fmt.Printf("[Logger] [Level: %d, Name: %s] -> Delete: deleting %v keys\n", cacher.GetRunInfo(ctx).Level(), l.next.Name(), keys)

	err := l.next.MDel(ctx, keys)

	duration := time.Since(startTime)
	if err != nil {
		fmt.Printf("[Logger] [Level: %d, Name: %s] <- Delete: finished in %s with error: %v\n", cacher.GetRunInfo(ctx).Level(), l.next.Name(), duration, err)
	} else {
		fmt.Printf("[Logger] [Level: %d, Name: %s] <- Delete: finished in %s successfully\n", cacher.GetRunInfo(ctx).Level(), l.next.Name(), duration)
	}

	return err
}

func (l *loggerWrapper[K, V]) Name() string {
	return l.next.Name()
}
