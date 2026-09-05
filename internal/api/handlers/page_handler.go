package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/yugjain1212/crawliq/internal/api/response"
	"github.com/yugjain1212/crawliq/internal/service"
)

// PageHandler exposes single-page routes. Like CrawlHandler it stays
// thin: parse request, call service, translate result.
type PageHandler struct {
	svc *service.CrawlService
}

func NewPageHandler(svc *service.CrawlService) *PageHandler {
	return &PageHandler{svc: svc}
}

// GetPage handles GET /pages/:id. The response includes the full
// page record (HTML, title, status code, etc.).
func (h *PageHandler) GetPage(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	page, err := h.svc.GetPage(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPageNotFound) {
			response.NotFound(c, "page not found")
			return
		}
		log.Error().Err(err).Int64("page_id", id).Msg("failed to fetch page")
		response.InternalError(c, "could not fetch page")
		return
	}

	response.Success(c, http.StatusOK, page)
}