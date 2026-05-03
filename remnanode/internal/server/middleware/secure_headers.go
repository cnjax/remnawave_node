package middleware

import "github.com/gin-gonic/gin"

// SecureHeaders sets security-related response headers matching the defaults
// provided by the Node.js helmet middleware used in the TS implementation.
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-XSS-Protection", "0")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Strict-Transport-Security", "max-age=15552000; includeSubDomains")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")
		c.Next()
	}
}
