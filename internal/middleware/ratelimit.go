package middleware

import (
	"context"
	"desafio-pos-graduacao/internal/config"
	"desafio-pos-graduacao/internal/limiter"
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strings"
	"time"
)

type RateLimitMiddleware struct {
	l          *limiter.Limiter
	cfg        config.AppConfig
	next       http.Handler
	trustProxy bool
	apiKeyHdr  string
}

func NewRateLimitMiddleware(l *limiter.Limiter, cfg config.AppConfig, next http.Handler) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		l:          l,
		cfg:        cfg,
		next:       next,
		trustProxy: cfg.TrustProxy,
		apiKeyHdr:  cfg.APIKeyHeader,
	}
}

func (m *RateLimitMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ip := m.extractIP(r)
	token := strings.TrimSpace(r.Header.Get(m.apiKeyHdr))

	res, err := m.l.Allow(ctx, ip, token)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !res.Allowed {
		retrySec := int(math.Ceil(res.RetryAfter.Seconds()))
		w.Header().Set("Retry-After", strconvIotaSafe(retrySec))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)

		msg := "you have reached the maximum number of requests or actions allowed within a certain time frame"
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":       "too_many_requests",
			"message":     msg,
			"retry_after": res.RetryAfter.String(),
			"scope":       string(res.Scope),
			"key":         res.Key,
		})
		return
	}
	m.next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, "rate_limiter_scope", res.Scope)))
}

func (m *RateLimitMiddleware) extractIP(r *http.Request) string {
	if m.trustProxy {
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				ip := strings.TrimSpace(parts[0])
				if ip != "" {
					return ip
				}
			}
		}
		if xr := r.Header.Get("X-Real-IP"); xr != "" {
			return strings.TrimSpace(xr)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func strconvIotaSafe(i int) string {
	return strings.TrimSpace(strings.ReplaceAll((time.Duration(i) * time.Second).String(), "s", ""))
}
