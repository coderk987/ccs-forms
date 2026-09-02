package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware authenticates bearer tokens and stores the verified user ID
// in Gin's context for downstream handlers. It does not decide whether that
// user may access a particular form; each handler still enforces ownership or
// view permissions.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing authorization header",
			})
			c.Abort()
			return
		}

		// Fields tolerates surrounding whitespace but rejects missing or extra
		// token material.
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header",
			})
			c.Abort()
			return
		}

		userID, err := ParseUserID(parts[1])
		if errors.Is(err, ErrJWTSecretNotConfigured) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication is not configured"})
			c.Abort()
			return
		}
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		// The ID is set only after signature, algorithm, expiry, and subject
		// validation have all succeeded.
		c.Set("userID", userID)

		c.Next()
	}
}
