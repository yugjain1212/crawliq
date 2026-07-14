package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/yugjain1212/crawliq/internal/api/response"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().Interface("panic", err).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Bytes("stack", debug.Stack()).
					Msg("Recovered from panic")
				response.Error(c, http.StatusInternalServerError, response.CodeInternal, "An internal error occured.")
				c.Abort()
			}
		}()
		c.Next()

	}
}
