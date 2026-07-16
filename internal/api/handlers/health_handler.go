package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yugjain1212/crawliq/internal/api/response"
)

type HealthHandler struct {
	db *pgxpool.Pool
}

func NewHealthHandler(db *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{db: db}
}

type healthStatus struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		response.Success(c, http.StatusServiceUnavailable, healthStatus{
			Status:   "Unhealthy",
			Database: "Unreachable",
		})
		return
	}
	response.Success(c, http.StatusOK, healthStatus{
		Status:   "healthy",
		Database: "Connected",
	})
}
