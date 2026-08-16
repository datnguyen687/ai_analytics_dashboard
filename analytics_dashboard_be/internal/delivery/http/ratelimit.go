package http

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"analytics-dashboard-be/internal/domain"
	"analytics-dashboard-be/internal/service"
)

// RateLimitPerUser throttles a route per authenticated user (must run after
// RequireAuth). Over the limit → 429 RATE_LIMITED with a Retry-After header.
// This guards the AI /ask endpoint against spamming (Gemini cost/quota).
func RateLimitPerUser(limiter domain.RateLimiter, scope string, limit, windowSeconds int) gin.HandlerFunc {
	return func(c *gin.Context) {
		identity := "anon"
		if u, ok := c.Get(ctxUserKey); ok {
			identity = u.(service.AuthenticatedUser).Username
		}
		key := scope + ":" + identity

		allowed, retryAfter, err := limiter.Allow(c.Request.Context(), key, limit, windowSeconds)
		if err == nil && !allowed {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			fail(c, domain.ErrRateLimited)
			return
		}
		c.Next()
	}
}
