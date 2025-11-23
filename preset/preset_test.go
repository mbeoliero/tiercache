package preset

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type User struct {
	ID   int
	Name string
}

func mockFetchUser(ctx context.Context, id int) (User, error) {
	return User{ID: id, Name: "user"}, nil
}

func setupRedis(t *testing.T) redis.UniversalClient {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
}

func TestNewRedisCache(t *testing.T) {
	rdb := setupRedis(t)
	ctx := context.Background()

	cache := NewRedisCache[int, User](
		rdb,
		"user:",
		time.Minute,
		mockFetchUser,
	)

	// Test Get
	u, _, err := cache.Get(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, u.ID)
	assert.Equal(t, "user", u.Name)
}

func TestNewLocalAndRedisCache(t *testing.T) {
	rdb := setupRedis(t)
	ctx := context.Background()

	cache := NewLocalAndRedisCache[int, User](
		rdb,
		"user:",
		time.Minute,
		time.Second,
		mockFetchUser,
	)

	// Test Get
	u, _, err := cache.Get(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, u.ID)
	assert.Equal(t, "user", u.Name)
}
