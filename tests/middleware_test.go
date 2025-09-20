package tests

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/alicebob/miniredis/v2"
    "github.com/example/ratelimiter/internal/config"
    httphandlers "github.com/example/ratelimiter/internal/http"
    "github.com/example/ratelimiter/internal/limiter"
    "github.com/example/ratelimiter/internal/limiter/storage/redis"
    "github.com/example/ratelimiter/internal/middleware"
    goredis "github.com/redis/go-redis/v9"
)

func TestMiddlewareReturns429(t *testing.T) {
    mr, _ := miniredis.Run()
    defer mr.Close()

    rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
    store := redisstorage.New(rdb)

    cfg := config.AppConfig{
        Mode:       config.ModeIP,
        HTTPAddr:   ":0",
        TrustProxy: false,
        DefaultIPPolicy: config.Policy{
            RPS:           1,
            BlockDuration: 300 * time.Millisecond,
            Window:        1 * time.Second,
        },
        APIKeyHeader: "API_KEY",
    }
    rl := limiter.New(store, cfg)

    mux := http.NewServeMux()
    mux.HandleFunc("/", httphandlers.RootHandler)
    h := middleware.NewRateLimitMiddleware(rl, cfg, mux)

    req := httptest.NewRequest("GET", "http://example.com/", nil)
    req.RemoteAddr = "1.2.3.4:12345"
    w := httptest.NewRecorder()

    // 1st ok
    h.ServeHTTP(w, req)
    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }

    // 2nd blocked
    w2 := httptest.NewRecorder()
    h.ServeHTTP(w2, req)
    if w2.Code != http.StatusTooManyRequests {
        t.Fatalf("expected 429, got %d", w2.Code)
    }
    var body map[string]any
    _ = json.Unmarshal(w2.Body.Bytes(), &body)
    if body["message"] != "you have reached the maximum number of requests or actions allowed within a certain time frame" {
        t.Fatalf("unexpected message: %v", body["message"])
    }
}