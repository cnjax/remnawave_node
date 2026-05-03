package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/internal/errors"
)

// Recovery returns a middleware that recovers from panics
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("error", err).
					Str("stack", string(debug.Stack())).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Msg("panic recovered")

				errors.SendError(c, errors.ErrInternalServer)
				c.Abort()
			}
		}()

		c.Next()
	}
}
