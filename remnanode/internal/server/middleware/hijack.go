package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// CloseUnknownRoute is a NoRoute handler that hijacks and closes the connection,
// matching TS not-found-exception.filter behavior.
func CloseUnknownRoute(c *gin.Context) {
	hijackAndClose(c)
}

// hijackAndClose seizes the underlying TCP connection and closes it without
// writing any HTTP response, matching TS behavior on JWT/auth failures.
func hijackAndClose(c *gin.Context) {
	hj, ok := c.Writer.(http.Hijacker)
	if !ok {
		// ResponseWriter doesn't support hijack (e.g. HTTP/2); fall back to abort.
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		log.Warn().Err(err).Msg("hijack failed")
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	conn.Close()
	c.Abort()
}
