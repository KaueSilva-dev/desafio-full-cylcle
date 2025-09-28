package main

import (
	"log"
	"net/http"
	"time"

	"desafio-pos-graduacao/internal/config"
	httpHandlers "desafio-pos-graduacao/internal/http"
	"desafio-pos-graduacao/internal/limiter"
	redisstorage "desafio-pos-graduacao/internal/limiter/storage/redis"
	"desafio-pos-graduacao/internal/middleware"

	"github.com/go-redis/redis/v8"
)

func main() {
	cfg := config.Parse()

	rbd := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})
	store := redisstorage.New(rbd)

	rl := limiter.New(store, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/", httpHandlers.RootHandler)
	mux.HandleFunc("/health", httpHandlers.HealthHandler)

	handler := middleware.NewRateLimitMiddleware(rl, cfg, mux)

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: handler,
	}

	log.Printf("starting server on %s(mode=%s)\n", cfg.HTTPAddr, cfg.Mode)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
