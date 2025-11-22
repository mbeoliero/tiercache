package tiercache

import (
	"context"
	"sync"

	"github.com/mbeoliero/tiercache/cacher"
)

var optionsPool = sync.Pool{
	New: func() interface{} {
		return &cacheOpts{}
	},
}

type cacheOpts struct {
	shouldSkipLayer       func(ctx context.Context, info cacher.BaseInfo) bool
	shouldFallbackOnError func(ctx context.Context, info cacher.BaseInfo, err error) bool
}

type OptFunc func(*cacheOpts)

// WithShouldSkipLayer sets the rule for skipping a cache layer.
// When the shouldSkip function returns true, the corresponding cache layer will be skipped,
// and the query will proceed directly to the next layer.
//
// Note: Cache levels are 1-based (e.g., Level 1, Level 2...), not 0-based.
//
// Example: Skip Level 1 or a cache layer named "redis"
//
//	cache.Get(ctx, key, tiercache.WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
//	    // cacher.GetRunInfo(ctx).Level() gets the current level (starts from 1)
//	    return cacher.GetRunInfo(ctx).Level() == 1 || info.Name() == "redis"
//	}))
func WithShouldSkipLayer(shouldSkip func(ctx context.Context, info cacher.BaseInfo) bool) OptFunc {
	return func(opts *cacheOpts) {
		opts.shouldSkipLayer = shouldSkip
	}
}

// WithFallbackOnLayerError sets whether to fallback to the next layer when an error occurs in the current layer (e.g., Redis connection failure).
// The function should return true to indicate fallback (default behavior), or false to return the error immediately.
func WithFallbackOnLayerError(shouldFallback func(ctx context.Context, info cacher.BaseInfo, err error) bool) OptFunc {
	return func(opts *cacheOpts) {
		opts.shouldFallbackOnError = shouldFallback
	}
}

func defaultOpts() *cacheOpts {
	opt := optionsPool.Get().(*cacheOpts)
	opt.free()
	return opt
}

func (m *cacheOpts) free() {
	m.shouldSkipLayer = nil
	m.shouldFallbackOnError = nil
}
