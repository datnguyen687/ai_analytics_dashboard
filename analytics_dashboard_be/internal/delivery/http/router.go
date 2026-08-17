package http

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"analytics-dashboard-be/internal/domain"
	"analytics-dashboard-be/internal/service"
)

// NewRouter wires the routes and middleware. Read-only API, so every data route
// is a GET except the natural-language POST /ask. All /api/v1 data routes require
// a valid JWT carrying the dashboard:view claim; /auth/login is public.
// AskRateLimit configures the per-user throttle on the AI endpoint.
type AskRateLimit struct {
	Limiter       domain.RateLimiter
	Limit         int
	WindowSeconds int
}

func NewRouter(h *Handler, authH *AuthHandler, auth *service.AuthService, askRL AskRateLimit, maxBodyBytes, maxImportBytes int64, corsOrigins []string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	importMaxBytes = maxImportBytes

	// Reject oversized request bodies before any handler reads them. The CSV
	// import route is exempt — it enforces its own (larger) limit.
	r.Use(BodyLimit(maxBodyBytes, "/api/v1/admin/orders/import"))

	allowed := make(map[string]bool, len(corsOrigins))
	for _, o := range corsOrigins {
		allowed[o] = true
	}

	r.Use(cors.New(cors.Config{
		// Allow the explicitly configured origins, plus ANY localhost/127.0.0.1
		// origin so a dev frontend works on whatever port it lands on (3000/3001…).
		AllowOriginFunc: func(origin string) bool {
			return allowed[origin] ||
				strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "http://127.0.0.1:")
		},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	v1 := r.Group("/api/v1")

	// Public auth routes.
	v1.POST("/auth/login", authH.Login)

	// Protected data routes: valid JWT + dashboard:view claim (both roles have it).
	protected := v1.Group("")
	protected.Use(RequireAuth(auth), RequireClaim(domain.ClaimDashboardView))
	{
		protected.GET("/auth/me", authH.Me)
		protected.GET("/meta", h.Meta)
		protected.GET("/suggestions", h.Suggestions)
		protected.GET("/dashboard", h.Dashboard)
		protected.GET("/orders", h.Orders)
		protected.GET("/forecast", h.Forecast)
		// The AI endpoint is additionally rate-limited per user.
		protected.POST("/ask",
			RateLimitPerUser(askRL.Limiter, "ask", askRL.Limit, askRL.WindowSeconds),
			h.Ask)
	}

	// Admin-only routes: additionally require the admin:manage claim (ADMIN role).
	admin := v1.Group("/admin")
	admin.Use(RequireAuth(auth), RequireClaim(domain.ClaimAdminManage))
	{
		admin.GET("/users", authH.Users)
		admin.POST("/orders/import", h.ImportOrders)
	}
	return r
}
