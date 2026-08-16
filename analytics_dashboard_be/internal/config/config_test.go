package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("CORS_ORIGIN", "")
	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("port = %q, want 8080", cfg.Port)
	}
	if cfg.JWTTTLHours != 24 {
		t.Errorf("jwt ttl = %d, want 24", cfg.JWTTTLHours)
	}
	if cfg.AskRateLimit != 15 {
		t.Errorf("ask rate limit = %d, want 15", cfg.AskRateLimit)
	}
	if cfg.MaxBodyBytes != 64*1024 {
		t.Errorf("max body = %d, want 65536", cfg.MaxBodyBytes)
	}
	if cfg.MaxQuestionChars != 1000 {
		t.Errorf("max question = %d, want 1000", cfg.MaxQuestionChars)
	}
	if len(cfg.CORSOrigins) != 2 {
		t.Errorf("cors origins = %v, want 2 defaults", cfg.CORSOrigins)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("ASK_RATE_LIMIT", "5")
	t.Setenv("MAX_QUESTION_CHARS", "50")
	t.Setenv("CORS_ORIGIN", "http://a.com, http://b.com ")
	cfg := Load()

	if cfg.Port != "9999" {
		t.Errorf("port = %q", cfg.Port)
	}
	if cfg.AskRateLimit != 5 {
		t.Errorf("ask rate = %d", cfg.AskRateLimit)
	}
	if cfg.MaxQuestionChars != 50 {
		t.Errorf("max question = %d", cfg.MaxQuestionChars)
	}
	if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[0] != "http://a.com" {
		t.Errorf("cors = %v (should split+trim)", cfg.CORSOrigins)
	}
}

func TestEnvIntFallbackOnBadValue(t *testing.T) {
	t.Setenv("JWT_TTL_HOURS", "notanumber")
	if cfg := Load(); cfg.JWTTTLHours != 24 {
		t.Errorf("bad int should fall back to default, got %d", cfg.JWTTTLHours)
	}
}
