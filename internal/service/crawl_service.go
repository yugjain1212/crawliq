// Package service contains the orchestration layer that wires together
// the repositories, sitemap discovery, and worker pool into a single
// "crawl this URL" operation. Handlers stay thin and translate HTTP ↔
// service; storage stays thin and translates SQL ↔ models; everything
// in between lives here.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/yugjain1212/crawliq/config"
	"github.com/yugjain1212/crawliq/internal/models"
	"github.com/yugjain1212/crawliq/internal/sitemap"
	"github.com/yugjain1212/crawliq/internal/storage"
	"github.com/yugjain1212/crawliq/internal/workers"
)

// Sentinel errors the handler layer can branch on without inspecting
// underlying pgx/network errors directly.
var (
	ErrInvalidURL      = errors.New("invalid website URL")
	ErrCrawlNotFound   = errors.New("crawl not found")
	ErrPageNotFound    = errors.New("page not found")
	ErrCrawlRunning    = errors.New("crawl is already running")
)

// CrawlService is the public face of the orchestration layer.
type CrawlService struct {
	crawlRepo  *storage.CrawlRepository
	pageRepo   *storage.PageRepository
	discoverer *sitemap.Discoverer
	dispatcher *workers.Dispatcher
	cfg        config.CrawlerConfig

	// inflight guards against two simultaneous POST /crawls for the
	// same website hammering the same upstream at once. It maps a
	// normalized website URL to the crawl id currently running.
	inflight   map[string]int64
	inflightMu sync.Mutex
}

// NewCrawlService assembles the service from its dependencies. Each
// piece is constructed in main.go and injected here so this package
// has no package-level state and stays trivially testable.
func NewCrawlService(
	crawlRepo *storage.CrawlRepository,
	pageRepo *storage.PageRepository,
	discoverer *sitemap.Discoverer,
	dispatcher *workers.Dispatcher,
	cfg config.CrawlerConfig,
) *CrawlService {
	return &CrawlService{
		crawlRepo:  crawlRepo,
		pageRepo:   pageRepo,
		discoverer: discoverer,
		dispatcher: dispatcher,
		cfg:        cfg,
		inflight:   make(map[string]int64),
	}
}

// StartCrawlInput is the minimal payload needed to kick off a new
// crawl. The handler is responsible for validating the raw string
// before handing it to the service.
type StartCrawlInput struct {
	Website string
}

// StartCrawl validates the input, normalizes the URL, deduplicates
// against any currently in-flight crawl for the same website, persists
// a pending crawl row, then runs the actual crawl in a goroutine and
// returns immediately with the created crawl. The caller polls
// GET /crawls/:id to see when it finishes.
func (s *CrawlService) StartCrawl(ctx context.Context, in StartCrawlInput) (*models.Crawl, error) {
	normalized, err := normalizeWebsite(in.Website)
	if err != nil {
		return nil, err
	}

	s.inflightMu.Lock()
	if existing, ok := s.inflight[normalized]; ok {
		s.inflightMu.Unlock()
		// Return the existing crawl rather than starting a duplicate —
		// the client can poll it for progress.
		existingCrawl, err := s.crawlRepo.GetByID(ctx, existing)
		if err != nil {
			return nil, fmt.Errorf("looking up in-flight crawl: %w", err)
		}
		return existingCrawl, ErrCrawlRunning
	}

	crawl, err := s.crawlRepo.Create(ctx, normalized)
	if err != nil {
		s.inflightMu.Unlock()
		return nil, fmt.Errorf("creating crawl row: %w", err)
	}

	s.inflight[normalized] = crawl.ID
	s.inflightMu.Unlock()

	// Run the crawl in the background. We intentionally use a fresh
	// context (decoupled from the HTTP request) so the caller can
	// disconnect and the crawl still completes. The lifetime of the
	// crawl is bounded by s.cfg.RequestTimeout per page.
	go s.runCrawl(crawl.ID, normalized)

	return crawl, nil
}

// GetCrawl is a thin pass-through to the repository, translating the
// storage sentinel into the service-level one for the handler layer.
func (s *CrawlService) GetCrawl(ctx context.Context, id int64) (*models.Crawl, error) {
	c, err := s.crawlRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrCrawlNotFound) {
			return nil, ErrCrawlNotFound
		}
		return nil, err
	}
	return c, nil
}

// DeleteCrawl removes a crawl row (and its pages via FK cascade).
// Returns ErrCrawlNotFound if the id doesn't exist.
func (s *CrawlService) DeleteCrawl(ctx context.Context, id int64) error {
	if err := s.crawlRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, storage.ErrCrawlNotFound) {
			return ErrCrawlNotFound
		}
		return err
	}
	return nil
}

// GetCrawlPages returns a paginated slice of pages belonging to a
// crawl, plus the total page count for the client's pagination UI.
func (s *CrawlService) GetCrawlPages(ctx context.Context, crawlID int64, opts storage.PageListOptions) ([]*models.Page, int64, error) {
	// Make sure the crawl exists before listing — better error than
	// "0 results, total 0" for a typo'd crawl id.
	if _, err := s.crawlRepo.GetByID(ctx, crawlID); err != nil {
		if errors.Is(err, storage.ErrCrawlNotFound) {
			return nil, 0, ErrCrawlNotFound
		}
		return nil, 0, err
	}
	pages, total, err := s.pageRepo.GetByCrawlID(ctx, crawlID, opts)
	if err != nil {
		return nil, 0, err
	}
	return pages, total, nil
}

// GetPage looks up a single page by id.
func (s *CrawlService) GetPage(ctx context.Context, id int64) (*models.Page, error) {
	p, err := s.pageRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrPageNotFound) {
			return nil, ErrPageNotFound
		}
		return nil, err
	}
	return p, nil
}

// runCrawl is the background pipeline: discover URLs via sitemap, fan
// them out to the worker pool, persist results, and finalize the crawl
// row. Errors here are logged and reflected in the final crawl status
// — they never propagate back to the HTTP caller (who is long gone by
// then anyway).
func (s *CrawlService) runCrawl(crawlID int64, website string) {
	defer func() {
		s.inflightMu.Lock()
		delete(s.inflight, website)
		s.inflightMu.Unlock()
	}()

	log.Info().
		Int64("crawl_id", crawlID).
		Str("website", website).
		Msg("starting crawl")

	if err := s.crawlRepo.UpdateStatus(context.Background(), crawlID, models.CrawlStatusRunning); err != nil {
		log.Error().Err(err).Int64("crawl_id", crawlID).Msg("failed to mark crawl as running")
		s.markFailed(crawlID, "could not transition to running")
		return
	}

	discoveryCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	discovery, err := s.discoverer.Discover(discoveryCtx, website)
	if err != nil {
		log.Error().Err(err).Int64("crawl_id", crawlID).Msg("sitemap discovery failed")
		s.markFailed(crawlID, "sitemap discovery failed: "+err.Error())
		return
	}

	urls := discovery.URLs
	if len(urls) == 0 {
		// Fall back to crawling just the website root when we couldn't
		// find any sitemap — this makes the "POST /crawls for an
		// obscure site" experience still produce something useful.
		urls = []string{website}
	}

	// Cap at the configured maximum so a misconfigured sitemap can't
	// queue millions of jobs.
	if s.cfg.MaxPagesPerCrawl > 0 && len(urls) > s.cfg.MaxPagesPerCrawl {
		log.Warn().
			Int64("crawl_id", crawlID).
			Int("discovered", len(urls)).
			Int("max", s.cfg.MaxPagesPerCrawl).
			Msg("discovered more URLs than max_pages_per_crawl; truncating")
		urls = urls[:s.cfg.MaxPagesPerCrawl]
	}

	log.Info().
		Int64("crawl_id", crawlID).
		Int("url_count", len(urls)).
		Int("sitemaps_found", discovery.SitemapsFound).
		Msg("dispatching crawl jobs")

	results := s.dispatcher.Dispatch(context.Background(), crawlID, urls)

	s.persistAndFinalize(crawlID, results, len(discovery.Errors))
}

// persistAndFinalize converts worker results into Page models, bulk
// inserts them, then updates the parent crawl row with the final
// tally. A single bulk insert keeps the round-trip cost roughly O(1)
// in number of pages instead of O(n).
func (s *CrawlService) persistAndFinalize(crawlID int64, results []workers.Result, discoveryErrors int) {
	pages := make([]*models.Page, 0, len(results))
	success, failed := 0, 0

	for _, r := range results {
		if !r.Succeeded() {
			failed++
		} else {
			// An HTTP 4xx/5xx still counts as a "successful fetch"
			// (we got a response) but not a "successful page" — flag
			// anything outside 2xx as failed for the crawl's stats.
			if r.StatusCode != nil && *r.StatusCode >= 200 && *r.StatusCode < 300 {
				success++
			} else {
				failed++
			}
		}

		pages = append(pages, &models.Page{
			CrawlID:        r.Job.CrawlID,
			URL:            r.Job.URL,
			StatusCode:     r.StatusCode,
			ContentType:    r.ContentType,
			ContentLength:  r.ContentLength,
			ResponseTimeMs: r.ResponseTimeMs,
			Title:          r.Title,
			HTML:           r.HTML,
			Error:          r.Error,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := s.pageRepo.BulkInsert(ctx, pages); err != nil {
		log.Error().Err(err).Int64("crawl_id", crawlID).Msg("bulk insert of pages failed")
		// Even if we couldn't persist the pages, still mark the crawl
		// as completed with whatever counts we have — leaving it in
		// "running" forever would be worse than reporting a failed
		// terminal state.
	}

	status := models.CrawlStatusCompleted
	if failed > 0 && success == 0 {
		// Every single page failed to fetch — mark the whole crawl as
		// failed so dashboards can distinguish "all good" from "all
		// broken". Mixed results stay "completed" so partial successes
		// are visible.
		status = models.CrawlStatusFailed
	}

	if err := s.crawlRepo.Complete(ctx, crawlID, storage.CompleteStats{
		Status:      status,
		TotalPages:  len(results),
		Success:     success,
		FailedPages: failed,
	}); err != nil {
		log.Error().Err(err).Int64("crawl_id", crawlID).Msg("finalizing crawl failed")
		return
	}

	log.Info().
		Int64("crawl_id", crawlID).
		Int("total", len(results)).
		Int("success", success).
		Int("failed", failed).
		Int("discovery_errors", discoveryErrors).
		Str("status", string(status)).
		Msg("crawl complete")
}

// markFailed is a tiny helper for the early-exit paths where the
// crawl never even produced results (discovery failure, etc.). It
// mirrors the Complete path but doesn't try to write any page stats.
func (s *CrawlService) markFailed(crawlID int64, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.crawlRepo.Complete(ctx, crawlID, storage.CompleteStats{
		Status: models.CrawlStatusFailed,
	}); err != nil {
		log.Error().Err(err).Int64("crawl_id", crawlID).Msg("could not mark crawl as failed")
		return
	}
	log.Warn().Int64("crawl_id", crawlID).Str("reason", reason).Msg("crawl marked failed")
}

// normalizeWebsite trims whitespace, requires a scheme, and returns the
// canonical "<scheme>://<host>" form (no trailing slash, no path) so
// two clients asking to crawl "https://Example.com/" and
// "example.com" don't get duplicate inflight entries.
func normalizeWebsite(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: empty URL", ErrInvalidURL)
	}

	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("%w: missing scheme or host", ErrInvalidURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%w: only http and https are supported", ErrInvalidURL)
	}

	u.Path = ""
	u.RawQuery = ""
	return u.String(), nil
}