package middleware

import (
	"net"

	"github.com/gin-gonic/gin"

	"github.com/remnawave/remnanode/internal/errors"
)

// InternalOnly rejects requests whose TCP remote address is not localhost.
// Uses c.Request.RemoteAddr (not spoofable via headers) to obtain the real peer IP.
func InternalOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			errors.SendError(c, errors.ErrInternalOnlyAccess)
			c.Abort()
			return
		}

		ip := net.ParseIP(host)
		if ip == nil || (!ip.Equal(net.IPv4(127, 0, 0, 1)) && !ip.Equal(net.IPv6loopback)) {
			errors.SendError(c, errors.ErrInternalOnlyAccess)
			c.Abort()
			return
		}

		c.Next()
	}
}

// TokenAuth validates the ?token= query parameter against the expected value.
// On mismatch the connection is hijacked and closed without a response body.
func TokenAuth(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Query("token") != expectedToken {
			hijackAndClose(c)
			return
		}
		c.Next()
	}
}
