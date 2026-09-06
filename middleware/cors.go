package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware answers browser preflights and allows every origin.
//
// TEMPORARY: the CORS_ORIGINS allowlist is disabled. Credentials are still not
// allowed, which is what makes the "*" wildcard legal here — a browser will
// refuse to attach cookies to a cross-site request against a wildcard origin.
// Restore the allowlist before this reaches production.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, ngrok-skip-browser-warning")
		c.Header("Access-Control-Max-Age", "600")

		// Preflights never reach a handler.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
