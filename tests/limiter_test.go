package tests

import (
	"context"
	"testing"
	"time"

	"desafio-pos-graduacao/internal/config"
	"desafio-pos-graduacao/internal/limiter"
	redisstorage "desafio-pos-graduacao/internal/limiter/storage/redis"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/go-redis/redis/v8"
)

func newLimiterForTest(t *testing.T, cfg config.AppConfig) (*limiter.Limiter, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(func() {
		mr.Close()
	})
	rdb := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})
	store := redisstorage.New(rdb)
	rl := limiter.New(store, cfg)
	return rl, mr
}

func TestIPRateLimitBlockAndUnblock(t *testing.T) {
	cfg := config.AppConfig{
		Mode:       config.ModeIP,
		TrustProxy: false,
		DefaultIPPolicy: config.Policy{
			RPS:           2,
			BlockDuration: 300 * time.Millisecond,
			Window:        1 * time.Second,
		},
	}
	rl, mr := newLimiterForTest(t, cfg)
	ctx := context.Background()

	ip := "1.2.3.4"

	// 1st allowed
	if res, err := rl.Allow(ctx, ip, ""); err != nil || !res.Allowed {
		t.Fatalf("expected allowed 1st, got %v, err=%v", res, err)
	}
	// 2nd allowed
	if res, err := rl.Allow(ctx, ip, ""); err != nil || !res.Allowed {
		t.Fatalf("expected allowed 2nd, got %v, err=%v", res, err)
	}
	// 3rd blocked
	res, err := rl.Allow(ctx, ip, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Allowed {
		t.Fatalf("expected blocked on 3rd")
	}
	if res.RetryAfter <= 0 {
		t.Fatalf("expected positive retryAfter")
	}
	// Fast-forward time to simulate block expiration
	mr.FastForward(350 * time.Millisecond)
	res2, err := rl.Allow(ctx, ip, "")
	if err != nil {
		t.Fatal(err)
	}
	if !res2.Allowed {
		t.Fatalf("expected allowed after block expired")
	}
}

func TestTokenOverrideBeatsIP(t *testing.T) {
	cfg := config.AppConfig{
		Mode:       config.ModeBoth,
		TrustProxy: false,
		DefaultIPPolicy: config.Policy{
			RPS:           1,
			BlockDuration: 300 * time.Millisecond,
			Window:        1 * time.Second,
		},
		DefaultTokenPolicy: config.Policy{
			RPS:           3,
			BlockDuration: 300 * time.Millisecond,
			Window:        1 * time.Second,
		},
		TokenOverrides: map[string]config.Policy{
			"abc123": {RPS: 5, BlockDuration: 300 * time.Millisecond, Window: 1 * time.Second},
		},
	}
	rl, _ := newLimiterForTest(t, cfg)
	ctx := context.Background()
	ip := "9.9.9.9"
	token := "abc123"

	// Should use token policy (RPS=5) not IP (RPS=1)
	for i := 0; i < 5; i++ {
		res, err := rl.Allow(ctx, ip, token)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Allowed {
			t.Fatalf("request %d should be allowed under token policy", i+1)
		}
	}
	res, err := rl.Allow(ctx, ip, token)
	if err != nil {
		t.Fatal(err)
	}
	if res.Allowed {
		t.Fatalf("6th should be blocked under token policy")
	}
}
