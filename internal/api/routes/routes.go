// Package routes wires HTTP routes to their handlers. It intentionally
// contains no business logic, its job is registration so that looking
// at this file tells you the entire public API surface of
// CrawlIQ ai a glance

package routes

import (
	"github.com/gin-gonic/gin"
)

// Handlers bundles every handler struct the router needs. Contructed once
// in main.go (after services/ repositories are constructed) and passed to New() to register routes.
// this keeps routes.go decoupled from how handlers get their dependecies
// (DB pool, services, etc) it just need the finished handler instances

type Handlers struct {
	Health *handlers.HealthHandler
	Crawl  *handlers.CrawlHandler
	Page   *handlers.PageHandler
}

/*
new builds and returns a fully configures Gin engine : global middleware applied then every route
registored under its handle
*/
func New(h Handlers) *gin.Engine {
	router := gin.New()
	/*Global Middleware - order matters. Recovery must Wrap everything so a panic anywhere downstream (e.g. a nil pointer in a handler ) gets caught and turned into
	a 500 response instead of crashing the whole server process. */
	router.Use(middleware.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	registerHealthRoutes(router, h.Health)
	registerCrawlRoutes(router, h.Crawl)
	registerPageRoutes(router, h.Page)

	return router
}

/*
	registeHeathRoutes wires the liveness/readiness endpoint used by docker HealthChecks,

load balancers, or uptime monitors.
*/
func registerHealthRoutes(router *gin.Engine, h *handlers.HealthHandler) {
	router.GET("/health", h.Check)

}

// registerCrawlRoutes wires the crawl related endpoints to their handlers.
func registerCrawlRoutes(router *gin.Engine, h *handlers.CrawlHandler) {
	crawls := router.Group("/crawls")
	{
		crawls.POST("", h.StartCrawl)
		crawls.GET("/:id", h.GetCrawl)
		crawls.GET("/:id/pages", h.GetCrawlPages)
		crawls.GET("/:id", h.DeleteCrawl)
	}
}

// registerPageRoutes wires the page related endpoints to their handlers.
// it addressed by its own page ID not nested under a crawl
func registerPageRoutes(router *gin.Engine, h *handlers.PageHandler) {
	router.GET("/pages/:id", h.GetPage)
}
