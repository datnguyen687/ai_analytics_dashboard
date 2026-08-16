package cmd

import (
	"log"

	"github.com/spf13/cobra"

	"analytics-dashboard-be/internal/cache"
	"analytics-dashboard-be/internal/delivery/http"
	"analytics-dashboard-be/internal/domain"
	"analytics-dashboard-be/internal/repository/postgres"
	"analytics-dashboard-be/internal/service"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := postgres.Connect(cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer db.Close()

		var c domain.Cache
		var limiter domain.RateLimiter
		if rc, err := cache.NewRedisCache(cfg.RedisURL); err != nil {
			log.Printf("redis unavailable (%v) — running without cache; rate limiting is in-memory", err)
			c = cache.NoopCache{}
			limiter = cache.NewMemoryRateLimiter()
		} else {
			c = rc
			limiter = rc // Redis-backed, shared across instances
		}

		repo := postgres.NewOrderRepo(db)
		userRepo := postgres.NewUserRepo(db)
		meta, err := repo.Meta(cmd.Context())
		if err != nil {
			log.Printf("could not preload meta (%v) — is the DB seeded?", err)
		}
		if cfg.JWTSecret == "dev-insecure-change-me" {
			log.Printf("WARNING: using the default JWT secret — set JWT_SECRET in production")
		}
		authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTLHours)

		analyticsSvc := service.NewAnalyticsService(repo, c, cfg.CacheTTL)
		forecastSvc := service.NewForecastService(repo, c, cfg.CacheTTL)

		// Interpretation: Gemini when a key is configured (with rule-based
		// fallback), otherwise the deterministic rule interpreter.
		var interp domain.Interpreter = service.NewRuleInterpreter(meta)
		if cfg.GeminiAPIKey != "" {
			interp = service.NewGeminiInterpreter(cfg.GeminiAPIKey, cfg.GeminiModel, meta, interp)
			log.Printf("using Gemini interpreter (model %s)", cfg.GeminiModel)
		} else {
			log.Printf("no GEMINI_API_KEY — using rule-based interpreter")
		}
		askSvc := service.NewAskService(repo, forecastSvc, interp)

		handler := http.NewHandler(analyticsSvc, forecastSvc, askSvc, cfg.MaxQuestionChars)
		authHandler := http.NewAuthHandler(authSvc)
		askRL := http.AskRateLimit{
			Limiter:       limiter,
			Limit:         cfg.AskRateLimit,
			WindowSeconds: cfg.AskRateWindow,
		}
		router := http.NewRouter(handler, authHandler, authSvc, askRL, cfg.MaxBodyBytes, cfg.CORSOrigins)

		log.Printf("listening on :%s", cfg.Port)
		return router.Run(":" + cfg.Port)
	},
}
