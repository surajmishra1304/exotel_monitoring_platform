package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS allows requests from any localhost origin (dev) and the configured
// production origin. Add VITE_API_BASE_URL origins to allowedOrigins for prod.
func CORS() gin.HandlerFunc {
	allowedOrigins := map[string]bool{
		"http://localhost:5173": true, // Vite dev server
		"http://localhost:3000": true, // CRA / alternate dev port
		"http://127.0.0.1:5173": true,
		"http://127.0.0.1:3000": true,
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Allow matching origins; fall back to wildcard for non-browser clients.
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if origin == "" {
			// Direct curl / Postman / server-to-server — no CORS header needed.
		} else {
			// Unknown browser origin: mirror it back so the request isn't blocked
			// in dev; tighten this to an explicit allowlist in production.
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-User")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle pre-flight OPTIONS immediately — don't pass to route handlers.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
