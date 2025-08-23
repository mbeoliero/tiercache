package rediscache

import (
	"context"
	"errors"
	"time"

	"github.com/mbeoliero/tiercache/store"
	"github.com/mbeoliero/tiercache/utils"
	"github.com/redis/go-redis/v9"
)

type RedisCache[K comparable, V any] struct {
	cli    redis.UniversalClient
	ttl    time.Duration
	prefix string
	opt    *Option[K, V]
}

func NewRedisCache[K comparable, V any](cli redis.UniversalClient, ttl time.Duration) *RedisCache[K, V] {
	return &RedisCache[K, V]{
		cli: cli,
		ttl: ttl,
		opt: defaultOption[K, V](),
	}
}

func (r *RedisCache[K, V]) SetPrefix(prefix string) *RedisCache[K, V] {
	r.prefix = prefix
	return r
}

func (r *RedisCache[K, V]) SetCodec(codec Codec[V]) *RedisCache[K, V] {
	r.opt.Codec = codec
	return r
}

func (r *RedisCache[K, V]) SetLogger(logger Logger) *RedisCache[K, V] {
	r.opt.Logger = logger
	return r
}

func (r *RedisCache[K, V]) SetMiddleware(mws ...store.Middleware[K, V]) *RedisCache[K, V] {
	r.opt.Mws = append(r.opt.Mws, mws...)
	return r
}

func (r *RedisCache[K, V]) ToStore() store.Interface[K, V] {
	return store.WrapperStore(r, r.opt.Mws...)
}

func (r *RedisCache[K, V]) MGet(ctx context.Context, keys []K) (map[K]V, []K, error) {
	ret := make(map[K]V, len(keys))
	miss := make([]K, 0)
	if len(keys) == 0 {
		return ret, miss, nil
	}
	redisKeys := r.getRedisKeys(keys)
	if r.opt.Logger != nil {
		r.opt.Logger.CtxDebug(ctx, "[redis-cache] read data from redis keys=%v", redisKeys)
	}

	p := r.cli.Pipeline()
	for _, key := range redisKeys {
		p.Get(ctx, key)
	}
	execResult, err := p.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		if r.opt.Logger != nil {
			r.opt.Logger.CtxError(ctx, "[redis-cache] MGet exec pipeline failed. err=%v", err)
		}
		return nil, nil, err
	}

	for index, result := range execResult {
		value, err := result.(*redis.StringCmd).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			if r.opt.Logger != nil {
				r.opt.Logger.CtxError(ctx, "[redis-cache] get redis failed. err=%v", err)
			}
			continue
		}

		var entity V
		if err = r.opt.Codec.Unmarshal([]byte(value), &entity); err != nil {
			if r.opt.Logger != nil {
				r.opt.Logger.CtxError(ctx, "[redis-cache] unmarshall failed. val=%value,err=%value", value, err)
			}
			continue
		}
		ret[keys[index]] = entity
	}

	if r.opt.Logger != nil {
		r.opt.Logger.CtxDebug(ctx, "[redis-cache] read data from redis keys=%v. ret=%v", redisKeys, ret)
	}

	for _, key := range keys {
		if _, ok := ret[key]; !ok {
			miss = append(miss, key)
		}
	}
	return ret, miss, nil
}

func (r *RedisCache[K, V]) MSet(ctx context.Context, entities map[K]V) error {
	if len(entities) == 0 {
		return nil
	}
	p := r.cli.Pipeline()
	for key, entity := range entities {
		data, err := r.opt.Codec.Marshal(entity)
		if err != nil {
			return err
		}
		p.SetEx(ctx, r.getRedisKey(key), data, r.ttl)
	}
	results, err := p.Exec(ctx)
	if err != nil {
		if r.opt.Logger != nil {
			r.opt.Logger.CtxError(ctx, "[redis-cache] MSet exec pipeline failed. err=%v", err)
		}
		return err
	}
	for _, result := range results {
		if _, err = result.(*redis.StatusCmd).Result(); err != nil {
			if r.opt.Logger != nil {
				r.opt.Logger.CtxError(ctx, "[redis-cache] set result failed. err=%v", err)
			}
			return err
		}
	}
	if r.opt.Logger != nil {
		r.opt.Logger.CtxDebug(ctx, "[redis-cache] set to redis success. size=%v", len(entities))
	}
	return nil
}

func (r *RedisCache[K, T]) MDel(ctx context.Context, keys []K) error {
	if len(keys) == 0 {
		return nil
	}
	if err := r.cli.Del(ctx, r.getRedisKeys(keys)...).Err(); err != nil {
		if r.opt.Logger != nil {
			r.opt.Logger.CtxError(ctx, "[redis-cache] delete failed.keys=%v,err=%v", keys, err)
		}
		return err
	}

	if r.opt.Logger != nil {
		r.opt.Logger.CtxDebug(ctx, "[redis-cache] delete success.keys=%v", keys)
	}
	return nil
}

func (r *RedisCache[K, T]) getRedisKeys(keys []K) []string {
	ret := make([]string, 0, len(keys))
	for _, k := range keys {
		ret = append(ret, r.getRedisKey(k))
	}
	return ret
}

func (r *RedisCache[K, T]) getRedisKey(k K) string {
	return r.prefix + utils.ToString(k)
}
