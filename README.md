# Tiercache

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Tiercache is a flexible, multi-level caching library for Go. It helps improve application performance by orchestrating multiple caching layers, such as a fast in-memory cache and a larger distributed cache.

## Features

- **Multi-Level Caching:** Combine multiple cache stores (e.g., in-memory, Redis) into a single, cohesive cache.
- **Cache-Aside Pattern:** Automatically fetches data from your primary data source on a cache miss and back-populates the cache layers.
- **Extensible:** Use middleware to add custom logic like logging, metrics, or tracing to any cache store.
- **Generic:** Works with any comparable key type and any value type.
- **Simple API:** Easy-to-use interface for getting, setting, and deleting cache entries.

## Installation

```bash
go get github.com/mbeoliero/tiercache
```

## Presets (Quick Start)

Tiercache provides pre-configured cache patterns for common use cases in the `preset` package.

### 1. Distributed Cache (Redis + DB)

Ideal for most distributed applications requiring shared state.

```go
import "github.com/mbeoliero/tiercache/preset"

// ...

// Create a standardized Redis cache
userCache := preset.NewRedisCache[int, User](
    redisClient,       // Redis client
    "users:",          // Key prefix
    time.Hour,         // Redis TTL
    func(ctx context.Context, id int) (User, error) {
        return db.GetUser(id) // Simple fetcher function
    },
)

// Use it
user, err := userCache.Get(ctx, 1001)
```

### 2. Hybrid Cache (Local + Redis + DB)

Perfect for hot data (e.g., configurations, popular content) to reduce network latency and Redis load.

```go
// Create a local + Redis cache with local memory layer
configCache := preset.NewLocalAndRedisCache[string, Config](
    redisClient,
    "config:",
    time.Hour,         // Redis TTL
    5*time.Minute,     // Local Memory TTL
    func(ctx context.Context, key string) (Config, error) {
        return db.GetConfig(key)
    },
)
```

## Advanced Usage (Custom)

For more control, you can manually compose layers using the core API.

Here's a simple example of how to set up a two-level cache (in-memory and Redis) with a data source.

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mbeoliero/tiercache"
	"github.com/mbeoliero/tiercache/datasource"
	"github.com/mbeoliero/tiercache/localcache"
	"github.com/mbeoliero/tiercache/middleware"
	"github.com/mbeoliero/tiercache/rediscache"
	"github.com/redis/go-redis/v9"
)

// Your data model
type User struct {
	ID   int
	Name string
}

// A mock database function
func fetchUsersFromDB(ctx context.Context, keys []int) (map[int]User, error) {
	fmt.Println("Fetching from database for keys:", keys)
	results := make(map[int]User)
	for _, key := range keys {
		// In a real app, you would query your database here
		results[key] = User{ID: key, Name: fmt.Sprintf("User-%d", key)}
	}
	return results, nil
}

func main() {
	ctx := context.Background()

	// 1. Set up your cache stores
	// Level 1: In-memory cache with a 5-minute TTL
	localStore := localcache.NewLocalCache[int, User](5 * time.Minute)

	// Level 2: Redis cache with a 1-hour TTL
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	redisStore := rediscache.NewRedisCache[int, User](redisClient, 1*time.Hour).ToStore()

	// 2. Set up your data source
	ds := datasource.NewDataSource(fetchUsersFromDB)

	// 3. Create the multi-level cache
	cache := tiercache.NewMultiLevelCache[int, User](
		localStore,
		redisStore,
		ds,
	).Build()

	// --- Usage Example ---

	// First Get: Data is not in any cache, so it's fetched from the DB
	// and populates both Redis and the local in-memory cache.
	fmt.Println("--- First request ---")
	user, err := cache.Get(ctx, 123)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got user: %+v\n\n", user)

	// Second Get: Data is now in the local in-memory cache, so it's returned from there.
	// No database call is made.
	fmt.Println("--- Second request ---")
	user, err = cache.Get(ctx, 123)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Got user: %+v\n", user)
}
```

## Middleware

Tiercache supports middleware to add custom logic to any cache store. This is useful for tasks like logging, metrics, or tracing.

Here's how you can apply the built-in logger middleware:

```go
import "github.com/mbeoliero/tiercache/middleware"

// ... (inside your main function)

// Create a new cache and apply middleware
cacheWithLogger := tiercache.NewMultiLevelCache[int, User](
    localStore,
    redisStore,
    ds,
).Use(middleware.LoggerMiddleware[int, User]()).Build()


// Now, all operations on the cache will be logged
fmt.Println("--- Request with logging ---")
user, err := cacheWithLogger.Get(ctx, 456)
if err != nil {
    panic(err)
}
fmt.Printf("Got user: %+v\n", user)
```

This will produce detailed logs for each cache operation, helping you debug and monitor your cache's behavior.

## Options

### Skipping Layers

You can dynamically skip specific cache layers for a request using `WithShouldSkipLayer`. This is useful for cases like force-refreshing data from a lower level (e.g., skipping local cache to hit Redis or DB).

```go
import (
    "github.com/mbeoliero/tiercache"
    "github.com/mbeoliero/tiercache/cacher"
)

// ...

// Skip the first level (Level 1, e.g., local in-memory cache)
user, _, err := cache.Get(ctx, 123, tiercache.WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
    // Skip if Level is 1
    return cacher.GetRunInfo(ctx).Level() == 1
}))

// Or skip by name (if your store implements Name())
user, _, err = cache.Get(ctx, 123, tiercache.WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
    return info.Name() == "local-cache"
}))
```

## Testing

To run the project's tests:

```bash
go test ./...
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
