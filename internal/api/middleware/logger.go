package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

/*
Logger returns a Gin middleware that logs one structured line per
request: method, path, status code, client IP, and how long the
request took to handle. This runs for every request that reaches it
(after Recovery), success or failure alike.
*/
func logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		/* Let the actual handler run first — we log after, so we have
		the final status code and total duration to report.*/

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

		/*Surface any handler-level errors gin.Context accumulated via
		c.Error(...), so they show up in logs even if the handler
		already sent a response.*/
		for _, ginErr := range c.Errors {
			log.Error().
				Str("method", c.Request.Method).
				Str("path", path).
				Err(ginErr.Err).
				Msg("handler error")

		}
	}
}
