package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/yugjain1212/crawliq/internal/api/response"
	"github.com/yugjain1212/crawliq/internal/service"
	"github.com/yugjain1212/crawliq/internal/storage"
)

// CrawlHandler exposes the crawl-related routes documented in
// routes.go. It is intentionally thin: all it does is parse the HTTP
// request, hand off to the service layer, and translate the result
// (including service-level sentinel errors) into the right HTTP
// status + envelope.
type CrawlHandler struct {
	svc *service.CrawlService
}

func NewCrawlHandler(svc *service.CrawlService) *CrawlHandler {
	return &CrawlHandler{svc: svc}
}

// startCrawlRequest is the body shape for POST /crawls.
type startCrawlRequest struct {
	Website string `json:"website" binding:"required"`
}

// StartCrawl handles POST /crawls. It accepts a JSON body with a
// single "website" field, creates the crawl row, and returns the
// pending crawl. The crawl itself runs asynchronously in the service
// layer.
func (h *CrawlHandler) StartCrawl(c *gin.Context) {
	var req startCrawlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "request body must include a non-empty 'website' field")
		return
	}

	crawl, err := h.svc.StartCrawl(c.Request.Context(), service.StartCrawlInput{
		Website: req.Website,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			response.BadRequest(c, err.Error())
		case errors.Is(err, service.ErrCrawlRunning):
			// 202 Accepted: the request was understood and an
			// in-progress crawl is being reused.
			response.Success(c, http.StatusAccepted, gin.H{
				"message": "a crawl for this website is already in progress",
				"crawl":   crawl,
			})
			return
		default:
			log.Error().Err(err).Msg("failed to start crawl")
			response.InternalError(c, "could not start crawl")
			return
		}
	}

	// 201 Created — a new crawl row was actually created.
	response.Success(c, http.StatusCreated, crawl)
}

// GetCrawl handles GET /crawls/:id. Returns the current crawl state
// (pending / running / completed / failed) plus the live counters.
func (h *CrawlHandler) GetCrawl(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	crawl, err := h.svc.GetCrawl(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrCrawlNotFound) {
			response.NotFound(c, "crawl not found")
			return
		}
		log.Error().Err(err).Int64("crawl_id", id).Msg("failed to fetch crawl")
		response.InternalError(c, "could not fetch crawl")
		return
	}

	response.Success(c, http.StatusOK, crawl)
}

// GetCrawlPages handles GET /crawls/:id/pages with pagination via
// ?limit=N&offset=N. The response includes the page slice, total
// count, and the echo of limit/offset so the client can build its
// pagination UI without guessing.
func (h *CrawlHandler) GetCrawlPages(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	limit, err := parseIntQuery(c, "limit", 50, 1, 500)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	offset, err := parseIntQuery(c, "offset", 0, 0, 1_000_000)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	pages, total, err := h.svc.GetCrawlPages(c.Request.Context(), id, storage.PageListOptions{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		if errors.Is(err, service.ErrCrawlNotFound) {
			response.NotFound(c, "crawl not found")
			return
		}
		log.Error().Err(err).Int64("crawl_id", id).Msg("failed to fetch crawl pages")
		response.InternalError(c, "could not fetch crawl pages")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"pages":  pages,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// DeleteCrawl handles DELETE /crawls/:id. Returns 204 with no body on
// success.
func (h *CrawlHandler) DeleteCrawl(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.svc.DeleteCrawl(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrCrawlNotFound) {
			response.NotFound(c, "crawl not found")
			return
		}
		log.Error().Err(err).Int64("crawl_id", id).Msg("failed to delete crawl")
		response.InternalError(c, "could not delete crawl")
		return
	}

	c.Status(http.StatusNoContent)
}

// parseIDParam extracts a positive int64 path parameter. Used by every
// handler that takes :id so the error message stays consistent.
func parseIDParam(c *gin.Context, name string) (int64, error) {
	raw := c.Param(name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New(name + " must be a positive integer")
	}
	return id, nil
}

// parseIntQuery reads ?name=… with a default, min, and max. It returns
// an error if the value is missing (empty falls back to default),
// unparseable, or outside the allowed range.
func parseIntQuery(c *gin.Context, name string, def, min, max int) (int, error) {
	raw := c.Query(name)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New(name + " must be an integer")
	}
	if v < min || v > max {
		return 0, errors.New(name + " must be between " + strconv.Itoa(min) + " and " + strconv.Itoa(max))
	}
	return v, nil
}