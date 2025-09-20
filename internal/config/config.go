package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Mode string

const (
	ModeIP Mode = "ip"
	ModeToken Mode = "token"
	ModeBoth Mode = "both"
)

type Policy struct {
	RPS int
	BlockDuration time.Duration
	Window time.Duration
}

type TokenOverride struct {
	Token string `json:"token"`
	RPS int `json:"rps"`
	Block string `json:"block"`
}

type RedisConfig struct {
	Addr string
	Password string
	DB int
}

type AppConfig struct {
	Mode Mode
	TrustProxy bool
	HTTPAddr string
	Redis RedisConfig
	DefaultIPPolicy Policy
	DefaultTokenPolicy Policy
	TokenOverrides map[string]Policy
	APIKeyHeader string
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func musteParseDuration(name, s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(fmt.Errorf("invalid duration for %s: %v", name, err))
	}
	return d
}

func mustParseInt(name, s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Errorf("invalid integer for %s: %v", name, err))
	}
	return i
}

func Parse() AppConfig {
	modeStr := strings.ToLower(getEnv("MODE", "both"))
	var mode Mode
	switch Mode(modeStr) {
	case ModeIP, ModeToken, ModeBoth:
		mode = Mode(modeStr)
	default:
		panic(fmt.Errorf("invalid MODE: %s", modeStr))
	}

	windowsMs := mustParseInt("RATE_LIMIT_WINDOWS_MS", getEnv(("RATE_LIMIT_WINDOWS_MS"), "1000"))
	win := time.Duration(windowsMs) * time.Millisecond

	ipRps := mustParseInt(("RATE_LIMIT_IP_RPS"), getEnv("RATE_LIMIT_IP_RPS", "5"))
	ipBlock := musteParseDuration("RATE_LIMIT_IP_BLOCK", getEnv("RATE_LIMIT_IP_BLOCK", "5m"))

	tokenDefRps := mustParseInt("RATE_LIMIT_TOKEN_DEFAULT_RPS", getEnv("RATE_LIMIT_TOKEN_DEFAULT_RPS", "10"))
	tokenDefBlock := musteParseDuration("RATE_LIMIT_TOKEN_DEFAULT_BLOCK", getEnv("RATE_LIMIT_TOKEN_DEFAULT_BLOCK", "5m"))

	overridesJSON := getEnv("RATE_LIMIT_TOKENS_JSON", "[]")
	var ov []TokenOverride
	if err := json.Unmarshal([]byte(overridesJSON), &ov); err != nil {
		panic(fmt.Errorf("invalid RATE_LIMIT_TOKENS_JSON: %v", err))
	}

	ovMap := make(map[string]Policy)
	for _, o := range ov {
		if o.Token == ""{
			panic("override token cannot be empty")
		}
		ovMap[o.Token] = Policy{
			RPS: o.RPS,
			BlockDuration: musteParseDuration("override_block", o.Block),
			Window: win,
		}
	}
	
	db := mustParseInt("REDIS_DB", getEnv("REDIS_DB", "0"))

	trustProxy := strings.ToLower(getEnv("TRUST_PROXY", "true")) == "true"

	return AppConfig{
		Mode: mode,
		TrustProxy: trustProxy,
		HTTPAddr: getEnv("HTTP_ADDR", ":8080"),
		Redis : RedisConfig {
			Addr : getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB: db,
		},
		DefaultIPPolicy: Policy{
			RPS: ipRps,
			BlockDuration: ipBlock,
			Window: win,
		},
		DefaultTokenPolicy: Policy{
			RPS: tokenDefRps,
			BlockDuration: tokenDefBlock,
			Window: win,
		},
		TokenOverrides: ovMap,
		APIKeyHeader: getEnv("API_KEY_HEADER", "API_KEY"),
	}
}