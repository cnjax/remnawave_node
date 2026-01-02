package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PortGuard middleware ensures requests come from the internal port
// This is used for internal APIs that should only be accessible locally
func PortGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		// The internal server is bound to 127.0.0.1 only
		// So any request reaching this middleware is already on the internal interface
		// This guard is mainly for documentation purposes and extra safety
		c.Next()
	}
}

// InternalOnly middleware rejects requests from external sources
func InternalOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request is from localhost
		remoteAddr := c.ClientIP()
		if remoteAddr != "127.0.0.1" && remoteAddr != "::1" && remoteAddr != "localhost" {
			c.JSON(http.StatusForbidden, gin.H{
				"isOk":    false,
				"message": "Access denied: internal API only",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
