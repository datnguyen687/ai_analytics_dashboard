package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"analytics-dashboard-be/internal/domain"
)

// BodyLimit rejects oversized request bodies — a cheap guard against large
// payloads / attachments being posted at the API. It checks the declared
// Content-Length up front and also caps the actual read via MaxBytesReader
// (so a lying/absent Content-Length can't sneak a huge body through).
// Paths matching a skipPrefix are left untouched (e.g. the CSV import route,
// which applies its own, larger limit).
func BodyLimit(maxBytes int64, skipPrefixes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, p := range skipPrefixes {
			if strings.HasPrefix(c.Request.URL.Path, p) {
				c.Next()
				return
			}
		}
		if c.Request.ContentLength > maxBytes {
			fail(c, domain.ErrPayloadTooLarge)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
