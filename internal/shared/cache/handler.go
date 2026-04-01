package cache

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *CacheService
}

func NewHandler(service *CacheService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.metrics.Snapshot())
}

func (h *Handler) ResetMetrics(c *gin.Context) {
	h.service.metrics.Reset()
	c.JSON(http.StatusOK, gin.H{"message": "cache metrics reset"})
}
