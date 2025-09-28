package redisstorage

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisStorage struct {
	client     *redis.Client
	incrScript *redis.Script
}

func New(client *redis.Client) *RedisStorage {
	script := redis.NewScript(`
	if redis.call('EXISTS', KEYS[1]) == 1 and not tonumber(redis.call('GET', KEYS[1])) then
		return redis.error_reply("ERR value at key is not an integer")
	end
	local v = redis.call('INCR', KEYS[1])
	if v == 1 then 
		redis.call('PEXPIRE', KEYS[1], ARGV[1])
	end
	return v
	`)
	return &RedisStorage{
		client:     client,
		incrScript: script,
	}
}

func (r *RedisStorage) IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	ms := ttl.Milliseconds()
	res, err := r.incrScript.Run(ctx, r.client, []string{key}, ms).Result()
	if err != nil {
		return 0, err
	}
	switch v := res.(type) {
	case int64:
		return v, nil
	case string:
		return 0, nil
	default:
		return 0, nil
	}
}

func (r *RedisStorage) TTL(ctx context.Context, key string) (time.Duration, error) {
	d, err := r.client.PTTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return d, nil
}

func (r *RedisStorage) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *RedisStorage) SetNx(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, value, ttl).Result()
}

func (r *RedisStorage) Del(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
