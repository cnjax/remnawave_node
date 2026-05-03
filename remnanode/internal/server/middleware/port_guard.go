package middleware

import (
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// InternalOnly rejects requests whose TCP remote address is not localhost.
// Uses c.Request.RemoteAddr (not spoofable via headers) to obtain the real peer IP.
func InternalOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			sendForbidden(c)
			return
		}

		ip := net.ParseIP(host)
		if ip == nil || (!ip.Equal(net.IPv4(127, 0, 0, 1)) && !ip.Equal(net.IPv6loopback)) {
			sendForbidden(c)
			return
		}

		c.Next()
	}
}

func sendForbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"path":      c.Request.URL.Path,
		"message":   "Access denied: internal API only",
		"errorCode": "A004",
	})
	c.Abort()
}
