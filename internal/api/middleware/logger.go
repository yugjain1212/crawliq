package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		event := log.Info()
		if status >= http.StatusInternalServerError {
			event = log.Error()

		} else if status >= http.StatusBadRequest {
			event = log.Warn()

		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", status).
			Str("client_ip", c.ClientIP()).
			Dur("duration", duration).
			Int("response_size", c.Writer.Size()).
			Msg("HTTP request")

		for _, ginErr := range c.Errors {
			log.Error().
				Str("method", c.Request.Method).
				Str("path", path).
				Err(ginErr.Err).
				Msg("handler error")

		}
	}
}
