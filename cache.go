package tiercache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/mbeoliero/tiercache/cacher"
)

type LevelCache[K comparable, V any] struct {
	Store       cacher.Interface[K, V]
	Middlewares []cacher.Middleware[K, V]
}

type MultiLevelCache[K comparable, V any] struct {
	stores      []cacher.Interface[K, V]
	middlewares []cacher.Middleware[K, V]

	sync.RWMutex
	built atomic.Bool
}

func NewMultiLevelCache[K comparable, V any](stores ...cacher.Interface[K, V]) *MultiLevelCache[K, V] {
	return &MultiLevelCache[K, V]{
		stores: stores,
	}
}

func (c *MultiLevelCache[K, V]) Use(middleware cacher.Middleware[K, V]) *MultiLevelCache[K, V] {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

func (c *MultiLevelCache[K, V]) Build() *MultiLevelCache[K, V] {
	if c.built.Load() {
		return c
	}

	var finalStores []cacher.Interface[K, V]
	for i := 0; i < len(c.stores); i++ {
		finalStores = append(finalStores, cacher.WrapperStore(c.stores[i], c.middlewares...))
	}

	c.RWMutex.Lock()
	defer c.RWMutex.Unlock()
	c.stores = finalStores

	c.built.Store(true)
	return c
}

func (c *MultiLevelCache[K, V]) Get(ctx context.Context, key K, opts ...OptFunc) (V, bool, error) {
	ret, err := c.MGet(ctx, []K{key}, opts...)
	if err != nil {
		var zero V
		return zero, false, err
	}
	val, ok := ret[key]
	return val, ok, nil
}

func (c *MultiLevelCache[K, V]) MGet(ctx context.Context, keys []K, opts ...OptFunc) (map[K]V, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	o := defaultOpts()
	for _, opt := range opts {
		opt(o)
	}
	defer optionsPool.Put(o)

	//allFoundItems := make(map[K]V)
	//keysToFetch := keys
	//
	//for i := 0; i < len(c.stores); i++ {
	//	if len(keysToFetch) == 0 {
	//		break
	//	}
	//
	//	store := c.stores[i]
	//	foundItems, missingKeys, err := cacher.MGet(ctx, keysToFetch)
	//
	//	if err != nil {
	//		// return nil, fmt.Errorf("cache store idx[%d] name[%s] MGet error: %s", i, cacher.Name(), err)
	//		continue
	//	}
	//
	//	if len(foundItems) > 0 {
	//		for k, v := range foundItems {
	//			allFoundItems[k] = v
	//		}
	//
	//		// Back-population
	//		if i > 0 {
	//			for j := i - 1; j >= 0; j-- {
	//				_ = c.stores[j].MSet(ctx, foundItems)
	//			}
	//		}
	//	}
	//
	//	keysToFetch = missingKeys
	//}

	found, _, err := c.mGetRecursive(ctx, keys, 0, o)
	return found, err
}

func (c *MultiLevelCache[K, V]) Set(ctx context.Context, key K, value V, opts ...OptFunc) error {
	return c.MSet(ctx, map[K]V{key: value}, opts...)
}

func (c *MultiLevelCache[K, V]) MSet(ctx context.Context, entities map[K]V, opts ...OptFunc) error {
	if len(entities) == 0 {
		return nil
	}

	o := defaultOpts()
	for _, opt := range opts {
		opt(o)
	}
	defer optionsPool.Put(o)

	for i, source := range c.stores {
		// inject level info
		loopCtx := cacher.NewContext(ctx, cacher.NewRunInfo(i+1))
		if err := source.MSet(loopCtx, entities); err != nil {
			return fmt.Errorf("cache store idx[%d] MSet error: %s", i, err)
		}
	}

	return nil
}

func (c *MultiLevelCache[K, V]) Del(ctx context.Context, key K, opts ...OptFunc) error {
	return c.MDel(ctx, []K{key}, opts...)
}

func (c *MultiLevelCache[K, V]) MDel(ctx context.Context, keys []K, opts ...OptFunc) error {
	if len(keys) == 0 {
		return nil
	}

	o := defaultOpts()
	for _, opt := range opts {
		opt(o)
	}
	defer optionsPool.Put(o)

	for i, source := range c.stores {
		// inject level info
		loopCtx := cacher.NewContext(ctx, cacher.NewRunInfo(i+1))
		if err := source.MDel(loopCtx, keys); err != nil {
			return fmt.Errorf("cache store idx[%d] MDel error: %s", i, err)
		}
	}

	return nil
}

func (c *MultiLevelCache[K, V]) mGetRecursive(ctx context.Context, keys []K, levelIdx int, opts *cacheOpts) (map[K]V, []K, error) {
	if levelIdx >= len(c.stores) {
		return make(map[K]V), keys, nil // Return remaining keys as missing
	}

	// Inject level info into Context (only for Middleware awareness; even if index isn't passed via context, 
	// it's kept here for consistency with middleware)
	// TODO: Consider passing runInfo as a parameter to middleware to fully decouple from context
	mwCtx := cacher.NewContext(ctx, cacher.NewRunInfo(levelIdx+1))

	currentStore := c.stores[levelIdx]
	if opts.shouldSkipLayer != nil && opts.shouldSkipLayer(mwCtx, currentStore) {
		return c.mGetRecursive(ctx, keys, levelIdx+1, opts)
	}

	foundItems, missingKeys, err := currentStore.MGet(mwCtx, keys)
	if err != nil {
		// TODO: log error here
		// Check if we should fallback to the next layer
		shouldFallback := true // Default is to fallback
		if opts.shouldFallbackOnError != nil {
			shouldFallback = opts.shouldFallbackOnError(mwCtx, currentStore, err)
		}

		if !shouldFallback {
			// If configured not to fallback, return the error immediately
			return nil, nil, err
		}

		// Fallback: current layer failed, treat all keys as missing and proceed to the next layer
		return c.mGetRecursive(ctx, keys, levelIdx+1, opts)
	}

	if foundItems == nil {
		foundItems = make(map[K]V)
	}

	if len(missingKeys) > 0 {
		// Recursively query the next layer
		deeperItems, stillMissing, gErr := c.mGetRecursive(ctx, missingKeys, levelIdx+1, opts)
		if gErr != nil {
			return nil, nil, gErr
		}

		// Back-populate data
		if len(deeperItems) > 0 {
			for k, v := range deeperItems {
				foundItems[k] = v
			}
			// Asynchronously or synchronously back-populate the current layer
			_ = currentStore.MSet(mwCtx, deeperItems)
		}

		// Update the final missing keys
		missingKeys = stillMissing
	}

	return foundItems, missingKeys, nil
}
