package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime settings, sourced from environment variables so no
// secrets live in the repository.
// DefaultJWTSecret is the placeholder secret; the server refuses to start with it
// in production (see cmd/serve.go).
const DefaultJWTSecret = "dev-insecure-change-me"

type Config struct {
	Env          string // "development" (default) | "production"
	Port         string
	DatabaseURL  string
	RedisURL     string
	CacheTTL     int      // seconds; 0 disables caching
	CORSOrigins  []string
	SeedCSVPath  string
	GeminiAPIKey string   // if empty, /ask falls back to the rule-based interpreter
	GeminiModel  string
	JWTSecret    string
	JWTTTLHours  int
	AskRateLimit  int   // max /ask requests per window per user
	AskRateWindow int   // window length in seconds
	MaxBodyBytes  int64 // reject request bodies larger than this
	MaxQuestionChars int // reject /ask questions longer than this
}

func Load() Config {
	return Config{
		Env:         env("APP_ENV", "development"),
		Port:        env("PORT", "8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/analytics?sslmode=disable"),
		RedisURL:    env("REDIS_URL", "redis://localhost:6379/0"),
		CacheTTL:    envInt("CACHE_TTL_SECONDS", 120),
		CORSOrigins: splitCSV(env("CORS_ORIGIN", "http://localhost:3000,http://localhost:3001")),
		SeedCSVPath:  env("SEED_CSV_PATH", "data/mock_logistics_data.csv"),
		GeminiAPIKey: env("GEMINI_API_KEY", ""),
		GeminiModel:  env("GEMINI_MODEL", "gemini-3.5-flash"),
		JWTSecret:     env("JWT_SECRET", DefaultJWTSecret),
		JWTTTLHours:   envInt("JWT_TTL_HOURS", 24),
		AskRateLimit:     envInt("ASK_RATE_LIMIT", 15),
		AskRateWindow:    envInt("ASK_RATE_WINDOW_SECONDS", 60),
		MaxBodyBytes:     int64(envInt("MAX_BODY_BYTES", 64*1024)), // 64 KB
		MaxQuestionChars: envInt("MAX_QUESTION_CHARS", 1000),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
