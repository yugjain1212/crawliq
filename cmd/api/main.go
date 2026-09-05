// Package main is the CrawlIQ API entry point. It is deliberately
// small — its job is wiring, not logic. Everything interesting lives
// in the internal packages; this file just constructs them in the
// right order and starts / shuts down the HTTP server cleanly.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/yugjain1212/crawliq/config"
	"github.com/yugjain1212/crawliq/internal/api/handlers"
	"github.com/yugjain1212/crawliq/internal/api/routes"
	"github.com/yugjain1212/crawliq/internal/crawler"
	"github.com/yugjain1212/crawliq/internal/logger"
	"github.com/yugjain1212/crawliq/internal/service"
	"github.com/yugjain1212/crawliq/internal/sitemap"
	"github.com/yugjain1212/crawliq/internal/storage"
	"github.com/yugjain1212/crawliq/internal/workers"
)

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("crawliq terminated with error")
	}
}

// run keeps main()'s top-level shape trivially readable — load config,
// init logger, build dependencies, start the server, wait for a signal,
// shut down. All real error returns happen through run()'s error.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger.Init(cfg.Log)

	// Root context with cancellation tied to SIGINT / SIGTERM so the
	// graceful shutdown path below can stop accepting new requests and
	// drain in-flight ones cleanly.
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := storage.NewPostgresPool(rootCtx, &cfg.Database)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	log.Info().
		Str("name", "CrawlIQ").
		Str("env", os.Getenv("APP_ENV")).
		Msg("starting CrawlIQ API")

	// Construct dependencies. Order matters: repositories depend on the
	// pool, services depend on repositories, handlers depend on
	// services. Keep this section as a flat sequence — it's the entire
	// dependency graph at a glance.
	crawlRepo := storage.NewCrawlRepository(pool)
	pageRepo := storage.NewPageRepository(pool)

	fetcher := crawler.NewFetcher(cfg.Crawler.RequestTimeout, cfg.Crawler.MaxRedirects, cfg.Crawler.UserAgent)
	crwlr := crawler.NewCrawler(fetcher)

	workerPool := workers.NewPool(cfg.Crawler.Workers, cfg.Crawler.RequestTimeout, crwlr.ProcessJob)
	dispatcher := workers.NewDispatcher(workerPool)

	discoverer := sitemap.NewDiscoverer(cfg.Crawler.RequestTimeout, cfg.Crawler.UserAgent)

	crawlSvc := service.NewCrawlService(crawlRepo, pageRepo, discoverer, dispatcher, cfg.Crawler)

	// Set Gin to release mode in production for the perf + log-noise
	// benefits. main.go is the only place that should ever set this.
	if cfg.Server.Mode != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	h := routes.Handlers{
		Health: handlers.NewHealthHandler(pool),
		Crawl:  handlers.NewCrawlHandler(crawlSvc),
		Page:   handlers.NewPageHandler(crawlSvc),
	}
	router := routes.New(h)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run the server in its own goroutine so we can wait on the signal
	// context in main and trigger a graceful shutdown.
	serverErr := make(chan error, 1)
	go func() {
		log.Info().Int("port", cfg.Server.Port).Msg("listening for HTTP requests")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("http server error: %w", err)
	case <-rootCtx.Done():
		log.Info().Msg("shutdown signal received, draining...")
	}

	// Give in-flight requests up to 30 seconds to finish, then bail.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	log.Info().Msg("server shut down cleanly")
	return nil
}