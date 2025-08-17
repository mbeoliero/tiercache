package tiercache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type LevelCache[K comparable, V any] struct {
	Store       CacheStore[K, V]
	Middlewares []Middleware[K, V]
}

type MultiLevelCache[K comparable, V any] struct {
	stores      []CacheStore[K, V]
	middlewares []Middleware[K, V]

	sync.RWMutex
	built atomic.Bool
}

func NewMultiLevelCache[K comparable, V any](stores ...CacheStore[K, V]) *MultiLevelCache[K, V] {
	return &MultiLevelCache[K, V]{
		stores: stores,
	}
}

func (c *MultiLevelCache[K, V]) Use(middleware Middleware[K, V]) *MultiLevelCache[K, V] {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

func (c *MultiLevelCache[K, V]) Build() *MultiLevelCache[K, V] {
	if c.built.Load() {
		return c
	}

	var finalStores []CacheStore[K, V]
	for i := 0; i < len(c.stores); i++ {
		store := c.stores[i]
		wrapStore := store
		for j := len(c.middlewares) - 1; j >= 0; j-- {
			wrapStore = c.middlewares[j](wrapStore)
		}

		finalStores = append(finalStores, wrapStore)
	}

	c.RWMutex.Lock()
	defer c.RWMutex.Unlock()
	c.stores = finalStores

	c.built.Store(true)
	return c
}

func (c *MultiLevelCache[K, V]) Get(ctx context.Context, key K, opts ...OptFunc) (V, error) {
	ret, err := c.MGet(ctx, []K{key}, opts...)
	if err != nil {
		var zero V
		return zero, err
	}
	return ret[key], nil
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
	//	foundItems, missingKeys, err := store.MGet(ctx, keysToFetch)
	//
	//	if err != nil {
	//		// return nil, fmt.Errorf("cache store idx[%d] name[%s] MGet error: %s", i, store.Name(), err)
	//		continue
	//	}
	//
	//	if len(foundItems) > 0 {
	//		for k, v := range foundItems {
	//			allFoundItems[k] = v
	//		}
	//
	//		// 回种（Back-population）
	//		if i > 0 {
	//			for j := i - 1; j >= 0; j-- {
	//				_ = c.stores[j].MSet(ctx, foundItems)
	//			}
	//		}
	//	}
	//
	//	keysToFetch = missingKeys
	//}

	return c.mGetRecursive(ctx, keys, o.getLevel)
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
		if err := source.MSet(ctx, entities); err != nil {
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
		if err := source.MDel(ctx, keys); err != nil {
			return fmt.Errorf("cache store idx[%d] MDel error: %s", i, err)
		}
	}

	return nil
}

func (c *MultiLevelCache[K, V]) mGetRecursive(ctx context.Context, keys []K, levelIndex int) (map[K]V, error) {
	if levelIndex >= len(c.stores) {
		return make(map[K]V), nil
	}

	currentStore := c.stores[levelIndex]
	foundItems, missingKeys, err := currentStore.MGet(ctx, keys)
	if err != nil {
		//fmt.Printf("Error getting from cache %s: %v. Treating as miss and trying next level.\n", currentStore.Name(), err)
		return c.mGetRecursive(ctx, keys, levelIndex+1)
	}

	if foundItems == nil {
		foundItems = make(map[K]V)
	}

	if len(missingKeys) > 0 {
		deeperItems, gErr := c.mGetRecursive(ctx, missingKeys, levelIndex+1)
		if gErr != nil {
			return nil, gErr
		}

		for k, v := range deeperItems {
			foundItems[k] = v
		}

		if len(deeperItems) > 0 {
			_ = currentStore.MSet(ctx, deeperItems)
		}
	}

	return foundItems, nil
}
