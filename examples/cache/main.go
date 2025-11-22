package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mbeoliero/tiercache"
	"github.com/mbeoliero/tiercache/cacher"
	"github.com/mbeoliero/tiercache/datasource"
	"github.com/mbeoliero/tiercache/localcache"
	"github.com/mbeoliero/tiercache/middleware"
	"github.com/mbeoliero/tiercache/rediscache"
	"github.com/redis/go-redis/v9"
)

// User is the data model
type User struct {
	ID   int
	Name string
}

// fetchUsersFromDB simulates a database fetch
func fetchUsersFromDB(ctx context.Context, keys []int) (map[int]User, error) {
	fmt.Printf("Fetching from database for keys: %v\n", keys)
	results := make(map[int]User)
	for _, key := range keys {
		// Simulate DB latency
		time.Sleep(10 * time.Millisecond)
		results[key] = User{ID: key, Name: fmt.Sprintf("User-%d", key)}
	}
	return results, nil
}

func main() {
	ctx := context.Background()

	// 1. Set up your cache stores
	// Level 1: In-memory cache with a 1-minute TTL
	localStore := localcache.NewLocalCache[int, User](1 * time.Minute)

	// Level 2: Redis cache with a 5-minute TTL
	// Make sure you have a Redis server running at localhost:6379
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	redisStore := rediscache.NewRedisCache[int, User](redisClient, 5*time.Minute).ToStore()

	// 2. Set up your data source
	// The data source is treated as the final "cache" layer that always has the data (if it exists)
	ds := datasource.NewDataSource(fetchUsersFromDB)

	// 3. Create the multi-level cache
	// Order matters: L1 -> L2 -> DataSource
	cache := tiercache.NewMultiLevelCache[int, User](
		localStore,
		redisStore,
		ds,
	).
		Use(middleware.LoggerMiddleware[int, User]()). // Add logging middleware
		Build()

	// --- Usage Example ---

	// Scenario 1: Cache Miss (Fetches from DB, populates Redis and Local)
	fmt.Println("\n--- Request 1 (Cache Miss) ---")
	userID := 101
	user, _, err := cache.Get(ctx, userID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Result: %+v\n", user)

	// Scenario 2: Cache Hit (L1 - Local Memory)
	// Should be instant and no DB/Redis logs if logging level allows
	fmt.Println("\n--- Request 2 (L1 Cache Hit) ---")
	user, _, err = cache.Get(ctx, userID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Result: %+v\n", user)

	// Scenario 3: Simulate L1 Expiration / Deletion
	fmt.Println("\n--- Request 3 (Simulate L1 Miss -> L2 Hit) ---")
	// Manually remove from local cache to simulate L1 miss
	// Note: In real usage, this happens via TTL. Here we force it for demonstration.
	_ = localStore.MDel(ctx, []int{userID})

	user, _, err = cache.Get(ctx, userID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Result: %+v\n", user)

	// Scenario 4: Multi-Get
	fmt.Println("\n--- Request 4 (Multi-Get) ---")
	ids := []int{101, 102, 103}
	users, err := cache.MGet(ctx, ids)
	if err != nil {
		panic(err)
	}
	for id, u := range users {
		fmt.Printf("ID: %d, User: %s\n", id, u.Name)
	}

	// Scenario 5: Skip Layer
	fmt.Println("\n--- Request 5 (Skip Layer 1) ---")
	// Skip Level 1 (Local Cache) and fetch directly from Redis (or DB if missing in Redis)
	user, _, err = cache.Get(ctx, userID, tiercache.WithShouldSkipLayer(func(ctx context.Context, info cacher.BaseInfo) bool {
		// Skip if current layer is Level 1 (Local Cache)
		return cacher.GetRunInfo(ctx).Level() == 1
	}))
	if err != nil {
		panic(err)
	}
	fmt.Printf("Result: %+v\n", user)
}
