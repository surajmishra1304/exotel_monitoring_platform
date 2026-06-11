package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestID injects a unique X-Request-ID header into every request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = fmt.Sprintf("REQ-%d", time.Now().UnixNano())
		}
		c.Set("request_id", reqID)
		c.Header("X-Request-ID", reqID)
		c.Next()
	}
}
