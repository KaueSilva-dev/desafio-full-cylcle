package storage

import (
	"time"
	"context"
)

type Storage interface {
	IncrWithTTL(ctx context.Context, key string, ttl time.Duration)(int64, error)
	TTL(ctx context.Context, key string)(time.Duration, error)
	Exists(ctx context.Context, key string)(bool, error)
	SetNx(ctx context.Context, key string, value string, ttl time.Duration)(bool, error)
	Del(ctx context.Context, key string)error
}