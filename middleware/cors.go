package middleware

import (
	"net/http"
	"os"
	"strings"

	"ccs-forms/config"

	"github.com/gin-gonic/gin"
)

// devOrigins are the local dev servers allowed when CORS_ORIGINS is unset.
// They only apply in dev; a production deployment must list its origins.
var devOrigins = []string{
	"http://localhost:3000",
	"http://127.0.0.1:3000",
	"http://localhost:5173",
	"http://127.0.0.1:5173",
}

// allowedOrigins reads the CORS_ORIGINS allowlist once at startup. Origins are
// compared exactly (scheme + host + port), so an ngrok or deployed frontend
// must be listed in full, for example
// "https://ccs.example.com,https://ccs-dev.ngrok-free.app".
func allowedOrigins() map[string]struct{} {
	raw := strings.Split(os.Getenv("CORS_ORIGINS"), ",")
	allowed := make(map[string]struct{}, len(raw))
	for _, origin := range raw {
		if origin = strings.TrimRight(strings.TrimSpace(origin), "/"); origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	if len(allowed) == 0 && config.IsDev() {
		for _, origin := range devOrigins {
			allowed[origin] = struct{}{}
		}
	}
	return allowed
}

// CORSMiddleware answers browser preflights and echoes back only origins on the
// allowlist. The API authenticates with bearer tokens rather than cookies, so
// credentials are deliberately not allowed: an allowlisted origin still cannot
// make the browser attach the OAuth state cookie to a cross-site XHR.
func CORSMiddleware() gin.HandlerFunc {
	allowed := allowedOrigins()

	return func(c *gin.Context) {
		// Vary is set for every response, allowed or not, so shared caches never
		// serve one origin's CORS headers to another.
		c.Header("Vary", "Origin")

		origin := strings.TrimRight(c.GetHeader("Origin"), "/")
		if _, ok := allowed[origin]; ok && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, ngrok-skip-browser-warning")
			c.Header("Access-Control-Max-Age", "600")
		}

		// Preflights never reach a handler; an origin that was not allowlisted
		// gets a bare 204 the browser will reject, which is the intended answer.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
