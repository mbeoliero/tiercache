package preset

import (
	"time"

	"github.com/mbeoliero/tiercache"
	"github.com/mbeoliero/tiercache/cacher"
	"github.com/mbeoliero/tiercache/datasource"
	"github.com/mbeoliero/tiercache/localcache"
	"github.com/mbeoliero/tiercache/rediscache"
	"github.com/redis/go-redis/v9"
)

// NewRedisCache creates a two-level cache: Redis -> DataSource (DB).
// This is the standard pattern for distributed systems where consistency and shared state are priorities.
func NewRedisCache[K comparable, V any](
	client redis.UniversalClient,
	prefix string,
	ttl time.Duration,
	fetcher datasource.Fetcher[K, V],
	mws ...cacher.Middleware[K, V],
) *tiercache.MultiLevelCache[K, V] {

	// L1: Redis Cache
	redisStore := rediscache.NewRedisCache[K, V](client, ttl).
		SetPrefix(prefix).
		ToStore()

	// L2: Data Source (DB)
	// Convert the single-key fetcher to a batch fetcher automatically
	ds := datasource.NewDataSourceWithFetcher[K, V](fetcher)

	c := tiercache.NewMultiLevelCache[K, V](
		redisStore,
		ds,
	)
	for _, m := range mws {
		c = c.Use(m)
	}
	return c.Build()
}

// NewLocalAndRedisCache creates a three-level cache: Local Memory -> Redis -> DataSource (DB).
// This pattern is ideal for hot data, significantly reducing network I/O and Redis load
// by caching frequently accessed items in the application's local memory.
func NewLocalAndRedisCache[K comparable, V any](
	client redis.UniversalClient,
	prefix string,
	redisTTL time.Duration,
	localTTL time.Duration,
	fetcher datasource.Fetcher[K, V],
	mws ...cacher.Middleware[K, V],
) *tiercache.MultiLevelCache[K, V] {

	// L1: Local Memory Cache
	localStore := localcache.NewLocalCache[K, V](localTTL)

	// L2: Redis Cache
	redisStore := rediscache.NewRedisCache[K, V](client, redisTTL).
		SetPrefix(prefix).
		ToStore()

	// L3: Data Source (DB)
	ds := datasource.NewDataSourceWithFetcher[K, V](fetcher)

	c := tiercache.NewMultiLevelCache[K, V](
		localStore,
		redisStore,
		ds,
	)
	for _, m := range mws {
		c = c.Use(m)
	}
	return c.Build()
}
