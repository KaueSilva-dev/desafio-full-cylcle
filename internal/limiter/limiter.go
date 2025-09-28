package limiter

import (
	"context"
	"fmt"
	"time"

	"desafio-pos-graduacao/internal/config"
	"desafio-pos-graduacao/internal/limiter/storage"
)

type Scope string

const (
	scopeIp    Scope = "ip"
	scopeToken Scope = "token"
)

type Result struct {
	Allowed    bool
	RetryAfter time.Duration
	Scope      Scope
	Key        string
	Policy     config.Policy
}

type Limiter struct {
	store     storage.Storage
	cfg       config.AppConfig
	keyPrefix string
	now       func() time.Time
}

func New(store storage.Storage, cfg config.AppConfig) *Limiter {
	return &Limiter{
		store:     store,
		cfg:       cfg,
		keyPrefix: "rl:",
		now:       time.Now,
	}
}

func (l *Limiter) Allow(ctx context.Context, ip, token string) (Result, error) {
	scope, key, pol := l.resolvePolicy(ip, token)

	if pol.RPS <= 0 {
		return Result{Allowed: true, Scope: scope, Key: key, Policy: pol}, nil
	}

	blockKey := l.blockKey(scope, key)
	blocked, err := l.store.Exists(ctx, blockKey)
	if err != nil {
		return Result{}, err
	}

	if blocked {
		ttl, err := l.store.TTL(ctx, blockKey)
		if err != nil {
			return Result{}, err
		}
		if ttl < 0 {
			ttl = time.Second
		}
		return Result{
			Allowed:    false,
			RetryAfter: ttl,
			Scope:      scope,
			Key:        key,
			Policy:     pol,
		}, nil
	}

	now := l.now()
	sec := now.Unix()
	windowKey := l.windowKey(scope, key, sec)
	windowTTL := pol.Window + time.Second

	count, err := l.store.IncrWithTTL(ctx, windowKey, windowTTL)
	if err != nil {
		return Result{}, err
	}

	if count > int64(pol.RPS) {
		if _, err := l.store.SetNx(ctx, blockKey, "1", pol.BlockDuration); err == nil {
			// After setting the block, we can clear the window counter
			_ = l.store.Del(ctx, windowKey)
		}
		return Result{
			Allowed:    false,
			RetryAfter: pol.BlockDuration,
			Scope:      scope,
			Key:        key,
			Policy:     pol,
		}, nil
	}

	return Result{
		Allowed: true,
		Scope:   scope,
		Key:     key,
		Policy:  pol,
	}, nil
}

func (l *Limiter) resolvePolicy(ip, token string) (Scope, string, config.Policy) {
	switch l.cfg.Mode {
	case config.ModeIP:
		return scopeIp, ip, l.cfg.DefaultIPPolicy
	case config.ModeToken:
		if token != "" {
			if pol, ok := l.cfg.TokenOverrides[token]; ok {
				return scopeToken, token, pol
			}
			return scopeToken, token, l.cfg.DefaultTokenPolicy
		}
		return scopeToken, token, l.cfg.DefaultTokenPolicy
	case config.ModeBoth:
		if token != "" {
			if pol, ok := l.cfg.TokenOverrides[token]; ok {
				return scopeToken, token, pol
			}
			return scopeToken, token, l.cfg.DefaultTokenPolicy
		}
		return scopeIp, ip, l.cfg.DefaultIPPolicy
	default:
		return scopeIp, ip, l.cfg.DefaultIPPolicy
	}
}

func (l *Limiter) windowKey(scope Scope, key string, sec int64) string {
	return fmt.Sprintf("%s%s:%s:%d", l.keyPrefix, scope, key, sec)
}

func (l *Limiter) blockKey(scope Scope, key string) string {
	return fmt.Sprintf("%s%s:%sblocked", l.keyPrefix, scope, key)
}
