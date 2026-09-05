// Package routes wires HTTP routes to their handlers. It intentionally
// contains no business logic — its job is registration, so that looking
// at this file tells you the entire public API surface of CrawlIQ at
// a glance.
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/yugjain1212/crawliq/internal/api/handlers"
	"github.com/yugjain1212/crawliq/internal/api/middleware"
)

// Handlers carries every handler struct the router needs. Constructed
// once in main.go (after services / repositories are constructed) and
// passed to New() to register routes. This keeps routes.go decoupled
// from how handlers get their dependencies (DB pool, services, etc.) —
// it only needs the finished handler instances.
type Handlers struct {
	Health *handlers.HealthHandler
	Crawl  *handlers.CrawlHandler
	Page   *handlers.PageHandler
}

// New builds and returns a fully configured Gin engine: global
// middleware applied first, then every route registered under its
// handler.
func New(h Handlers) *gin.Engine {
	router := gin.New()

	// Global middleware — order matters. Recovery must wrap everything
	// so a panic anywhere downstream (e.g. a nil pointer in a handler)
	// gets caught and turned into a 500 response instead of crashing
	// the whole server process.
	router.Use(middleware.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	registerHealthRoutes(router, h.Health)
	registerCrawlRoutes(router, h.Crawl)
	registerPageRoutes(router, h.Page)

	return router
}

// registerHealthRoutes wires the liveness/readiness endpoint used by
// Docker health checks, load balancers, or uptime monitors.
func registerHealthRoutes(router *gin.Engine, h *handlers.HealthHandler) {
	router.GET("/health", h.Check)
}

// registerCrawlRoutes wires the crawl-related endpoints to their
// handlers. Path layout matches the README's API table:
//
//	POST   /crawls            — start a new crawl
//	GET    /crawls/:id        — fetch status + stats
//	GET    /crawls/:id/pages  — list the crawled pages
//	DELETE /crawls/:id        — delete the crawl + its pages
func registerCrawlRoutes(router *gin.Engine, h *handlers.CrawlHandler) {
	crawls := router.Group("/crawls")
	{
		crawls.POST("", h.StartCrawl)
		crawls.GET("/:id", h.GetCrawl)
		crawls.GET("/:id/pages", h.GetCrawlPages)
		crawls.DELETE("/:id", h.DeleteCrawl)
	}
}

// registerPageRoutes wires the single-page lookup. Lives at /pages/:id
// (not nested under a specific crawl) because a page already has a
// globally-unique primary key.
func registerPageRoutes(router *gin.Engine, h *handlers.PageHandler) {
	router.GET("/pages/:id", h.GetPage)
}