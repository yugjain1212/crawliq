package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

/*
		Cors return a Gin middleware that adds permissive Cross origin Resource Sharing header, allowing a
	    Browser based frontend running on different origin/port, e.g. localhost:3000 during develpment to call this
	    API Directly.

		CrawlIQ has no authenticated/user-specific data yet — every response
		is safe to expose to any origin. If auth or per-user data is added
		later, this should be tightened to an explicit allowlist of origins
		instead of "*"
*/
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE,,PATCH, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Writer.Header().Set("Access-Control-Max-Age", "3600")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return

		}
		c.Next()
	}
}
