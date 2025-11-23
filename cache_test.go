package tiercache

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/mbeoliero/tiercache/cacher"
	"github.com/mbeoliero/tiercache/middleware"
	"github.com/mbeoliero/tiercache/rediscache"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewMultiLevelCache(t *testing.T) {
	l1 := LocalCache{data: map[string]string{}}
	l2 := LocalCache{data: map[string]string{"1": "1"}}
	l3 := LocalCache{data: map[string]string{"2": "2", "1": "1"}}
	c := NewMultiLevelCache[string, string](l1, l2, l3)
	fmt.Println(c.MGet(context.TODO(), []string{"2"}))
	fmt.Printf("l1: %v\n", l1)
	fmt.Printf("l2: %v\n", l2)
	fmt.Printf("l3: %v\n", l3)
	fmt.Println(c.MGet(context.TODO(), []string{"1"}))
	fmt.Printf("l1: %v\n", l1)
	fmt.Printf("l2: %v\n", l2)
	fmt.Printf("l3: %v\n", l3)
}

type LocalCache struct {
	data map[string]string
	name string
	err  error
}

func (l LocalCache) Name() string {
	if l.name == "" {
		return "local_cache"
	}
	return l.name
}

func (l LocalCache) MSet(ctx context.Context, entities map[string]string) error {
	for k, v := range entities {
		l.data[k] = v
	}
	return l.err
}

func (l LocalCache) MGet(ctx context.Context, keys []string) (map[string]string, []string, error) {
	ret := make(map[string]string)
	miss := make([]string, 0)
	for _, k := range keys {
		if v, ok := l.data[k]; ok {
			ret[k] = v
		} else {
			miss = append(miss, k)
		}
	}
	return ret, miss, l.err
}

func (l LocalCache) MDel(ctx context.Context, keys []string) error {
	for _, k := range keys {
		delete(l.data, k)
	}
	return nil
}

func TestL1Cache(t *testing.T) {
	mld := NewMultiLevelCache[string, string](LocalCache{data: map[string]string{}, err: nil})
	v, err := mld.MGet(context.TODO(), []string{"1"})
	assert.Nil(t, err)
	assert.Equal(t, v["1"], "")

	mld = NewMultiLevelCache[string, string](LocalCache{data: map[string]string{"1": "1"}, err: nil})
	v, err = mld.MGet(context.TODO(), []string{"1"})
	assert.Nil(t, err)
	assert.Equal(t, v["1"], "1")
}

func TestL2Cache(t *testing.T) {
	//
	{
		l1 := LocalCache{data: map[string]string{}}
		l2 := LocalCache{data: map[string]string{"1": "1"}}
		mld := NewMultiLevelCache[string, string](l1, l2)
		v, err := mld.MGet(context.TODO(), []string{"2"})
		assert.Nil(t, err)
		assert.Equal(t, v["3"], "")

		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"1": "1"})
	}
	//
	{
		l1 := LocalCache{data: map[string]string{}}
		l2 := LocalCache{data: map[string]string{"1": "1"}}
		mld := NewMultiLevelCache[string, string](l1, l2)
		v, err := mld.MGet(context.TODO(), []string{"1"})
		assert.Nil(t, err)
		assert.Equal(t, v["1"], "1")

		assert.Equal(t, l1.data, map[string]string{"1": "1"})
		assert.Equal(t, l2.data, map[string]string{"1": "1"})
	}

	//
	{
		// Current layer error, should use data from upper layer
		l1 := LocalCache{data: map[string]string{}, err: errors.New("1")}
		l2 := LocalCache{data: map[string]string{"1": "1"}}
		mld := NewMultiLevelCache[string, string](l1, l2)
		v, err := mld.MGet(context.TODO(), []string{"1"})
		assert.Nil(t, err)
		assert.Equal(t, v["1"], "1")

		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"1": "1"})
	}
}

func TestL3Cache(t *testing.T) {
	{
		l1 := LocalCache{data: map[string]string{}}
		l2 := LocalCache{data: map[string]string{"2": "2"}}
		l3 := LocalCache{data: map[string]string{"3": "3"}}
		mld := NewMultiLevelCache[string, string](l1, l2, l3)
		v, err := mld.MGet(context.TODO(), []string{"1"}, WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
			return cacher.GetRunInfo(ctx).Level() == 1
		}))
		assert.Nil(t, err)
		assert.Equal(t, v["1"], "")

		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"2": "2"})
		assert.Equal(t, l3.data, map[string]string{"3": "3"})

		v, err = mld.MGet(context.TODO(), []string{"2"}, WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
			return cacher.GetRunInfo(ctx).Level() == 1
		}))
		assert.Nil(t, err)
		assert.Equal(t, v["2"], "2")

		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"2": "2"})
		assert.Equal(t, l3.data, map[string]string{"3": "3"})

		v, err = mld.MGet(context.TODO(), []string{"3"}, WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
			return cacher.GetRunInfo(ctx).Level() == 1
		}))
		assert.Nil(t, err)
		assert.Equal(t, v["3"], "3")

		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"2": "2", "3": "3"})
		assert.Equal(t, l3.data, map[string]string{"3": "3"})

		v, err = mld.MGet(context.TODO(), []string{"3"}, WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
			return cacher.GetRunInfo(ctx).Level() <= 2
		}))
		assert.Nil(t, err)
		assert.Equal(t, v["3"], "3")

		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"2": "2", "3": "3"})
		assert.Equal(t, l3.data, map[string]string{"3": "3"})

		err = mld.MDel(context.TODO(), []string{"3"})
		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"2": "2"})
		assert.Equal(t, l3.data, map[string]string{})
	}
}

func TestL4Cache(t *testing.T) {
	{
		l1 := LocalCache{data: map[string]string{}, name: "l1"}
		l2 := LocalCache{data: map[string]string{"2": "2"}, name: "l2"}
		l3 := LocalCache{data: map[string]string{"3": "3"}, name: "l3"}
		l4 := LocalCache{data: map[string]string{"4": "4"}, name: "l4"}
		mld := NewMultiLevelCache[string, string](l1, l2, l3, l4).Use(middleware.LoggerMiddleware[string, string]()).Build()
		v, err := mld.MGet(context.TODO(), []string{"1"}, WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
			return cacher.GetRunInfo(ctx).Level() == 1
		}))
		t.Log("=======")
		assert.Nil(t, err)
		assert.Equal(t, v["1"], "")

		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"2": "2"})
		assert.Equal(t, l3.data, map[string]string{"3": "3"})

		v, err = mld.MGet(context.TODO(), []string{"2"}, WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
			return cacher.GetRunInfo(ctx).Level() == 1
		}))
		t.Log("=======")
		assert.Nil(t, err)
		assert.Equal(t, v["2"], "2")

		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"2": "2"})
		assert.Equal(t, l3.data, map[string]string{"3": "3"})

		v, err = mld.MGet(context.TODO(), []string{"3"}, WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
			return cacher.GetRunInfo(ctx).Level() == 1
		}))
		t.Log("=======")
		assert.Nil(t, err)
		assert.Equal(t, v["3"], "3")

		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"2": "2", "3": "3"})
		assert.Equal(t, l3.data, map[string]string{"3": "3"})

		v, err = mld.MGet(context.TODO(), []string{"3"}, WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
			return cacher.GetRunInfo(ctx).Level() <= 2
		}))
		t.Log("=======")
		assert.Nil(t, err)
		assert.Equal(t, v["3"], "3")

		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"2": "2", "3": "3"})
		assert.Equal(t, l3.data, map[string]string{"3": "3"})

		v, err = mld.MGet(context.TODO(), []string{"3"}, WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
			return cacher.GetRunInfo(ctx).Level() <= 3
		}))
		t.Log("=======")
		assert.Nil(t, err)
		assert.Equal(t, v["3"], "")

		err = mld.MDel(context.TODO(), []string{"3"})
		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"2": "2"})
		assert.Equal(t, l3.data, map[string]string{})
		assert.Equal(t, l4.data, map[string]string{"4": "4"})

		v, err = mld.MGet(context.TODO(), []string{"4"}, WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
			return cacher.GetRunInfo(ctx).Level() <= 2
		}))
		t.Log("=======")
		assert.Nil(t, err)
		assert.Equal(t, v["4"], "4")

		assert.Equal(t, l1.data, map[string]string{})
		assert.Equal(t, l2.data, map[string]string{"2": "2"})
		assert.Equal(t, l3.data, map[string]string{"4": "4"})
		assert.Equal(t, l4.data, map[string]string{"4": "4"})

		v, err = mld.MGet(context.TODO(), []string{"4"})
		t.Log("=======")
		assert.Nil(t, err)
		assert.Equal(t, v["4"], "4")

		assert.Equal(t, l1.data, map[string]string{"4": "4"})
		assert.Equal(t, l2.data, map[string]string{"2": "2", "4": "4"})
		assert.Equal(t, l3.data, map[string]string{"4": "4"})
		assert.Equal(t, l4.data, map[string]string{"4": "4"})
	}
}

func TestSkipLayerComplex(t *testing.T) {
	l1 := LocalCache{data: map[string]string{}, name: "l1"}
	l2 := LocalCache{data: map[string]string{"key": "val2"}, name: "l2"}
	l3 := LocalCache{data: map[string]string{"key": "val3"}, name: "l3"}
	l4 := LocalCache{data: map[string]string{"key": "val4"}, name: "l4"}

	mld := NewMultiLevelCache[string, string](l1, l2, l3, l4)

	// Skip if Level is 2 AND Name is "l2"
	skipFunc := func(ctx context.Context, info cacher.BaseInfo) bool {
		return info.Name() == "l2"
	}

	v, err := mld.MGet(context.TODO(), []string{"key"}, WithShouldSkipLayer(skipFunc))
	assert.Nil(t, err)
	// Should skip L2 (val2) and hit L3 (val3)
	assert.Equal(t, "val3", v["key"])

	// L1 should be backfilled with val3
	assert.Equal(t, "val3", l1.data["key"])
	// L2 should be skipped and thus NOT backfilled (it has val2 anyway, but strictly it wasn't touched)
	assert.Equal(t, "val2", l2.data["key"])

	// Reset L1
	delete(l1.data, "key")

	// Condition that does NOT skip
	skipFunc2 := func(ctx context.Context, info cacher.BaseInfo) bool {
		return cacher.GetRunInfo(ctx).Level() == 2 && info.Name() == "other"
	}

	v2, err := mld.MGet(context.TODO(), []string{"key"}, WithShouldSkipLayer(skipFunc2))
	assert.Nil(t, err)
	// Should hit L2
	assert.Equal(t, "val2", v2["key"])
}

func TestLayerErrorFallback(t *testing.T) {
	// Case 1: Default behavior (fallback on error)
	l1 := LocalCache{data: map[string]string{}, err: errors.New("l1 error"), name: "l1"}
	l2 := LocalCache{data: map[string]string{"key": "value"}, name: "l2"}

	mld := NewMultiLevelCache[string, string](l1, l2)
	v, err := mld.MGet(context.TODO(), []string{"key"})
	assert.Nil(t, err)
	assert.Equal(t, "value", v["key"])

	// Case 2: Configured to NOT fallback on error
	l1 = LocalCache{data: map[string]string{}, err: errors.New("l1 error"), name: "l1"}
	l2 = LocalCache{data: map[string]string{"key": "value"}, name: "l2"}

	mld = NewMultiLevelCache[string, string](l1, l2)

	// Use WithFallbackOnLayerError to return false (do not fallback)
	v, err = mld.MGet(context.TODO(), []string{"key"}, WithFallbackOnLayerError(func(ctx context.Context, info cacher.BaseInfo, err error) bool {
		// Custom logic: if error is "l1 error", do not fallback
		return err.Error() != "l1 error"
	}))

	assert.NotNil(t, err)
	assert.Equal(t, "l1 error", err.Error())
	assert.Nil(t, v) // mGetRecursive returns nil map on error

	// Case 3: Configured to fallback on specific error
	l1 = LocalCache{data: map[string]string{}, err: errors.New("recoverable error"), name: "l1"}
	l2 = LocalCache{data: map[string]string{"key": "value"}, name: "l2"}

	mld = NewMultiLevelCache[string, string](l1, l2)

	v, err = mld.MGet(context.TODO(), []string{"key"}, WithFallbackOnLayerError(func(ctx context.Context, info cacher.BaseInfo, err error) bool {
		// Custom logic: fallback only if error is "recoverable error"
		return err.Error() == "recoverable error"
	}))

	assert.Nil(t, err)
	assert.Equal(t, "value", v["key"])
}

type localMapCache struct {
	data map[string]int
}

func (l *localMapCache) MGet(ctx context.Context, keys []string) (map[string]int, []string, error) {
	ret := make(map[string]int)
	miss := make([]string, 0)
	for _, k := range keys {
		if v, ok := l.data[k]; ok {
			ret[k] = v
		} else {
			miss = append(miss, k)
		}
	}
	return ret, miss, nil
}

func (l *localMapCache) MSet(ctx context.Context, entities map[string]int) error {
	for k, entity := range entities {
		l.data[k] = entity
	}
	return nil
}

func (l *localMapCache) MDel(ctx context.Context, keys []string) error {
	for _, key := range keys {
		delete(l.data, key)
	}
	return nil
}

func (l *localMapCache) Name() string {
	return "local-map-cache"
}

func TestRedisMapCache(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()
	rdb := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	l2 := rediscache.NewRedisCache[string, int](rdb, time.Hour).SetPrefix("pre:").SetMiddleware(rediscache.MetricsMiddleware[string, int]("test")).ToStore()
	l1 := &localMapCache{data: map[string]int{
		"k1": 1,
		"k2": 2,
		"k3": 3,
	}}

	ds := NewMultiLevelCache[string, int](l2, l1)
	ctx := context.TODO()

	v1, err := ds.MGet(ctx, []string{"k5", "k6"})
	assert.Nil(t, err)
	fmt.Println(v1)

	v2, err := ds.MGet(ctx, []string{"k3", "k4"})
	assert.Nil(t, err)
	fmt.Println(v2)

	l2.MGet(ctx, []string{"k1", "k2", "k3", "k4", "k5", "k6"})
}
