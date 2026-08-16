package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"analytics-dashboard-be/internal/domain"
)

// BodyLimit rejects oversized request bodies — a cheap guard against large
// payloads / attachments being posted at the API. It checks the declared
// Content-Length up front and also caps the actual read via MaxBytesReader
// (so a lying/absent Content-Length can't sneak a huge body through).
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			fail(c, domain.ErrPayloadTooLarge)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
