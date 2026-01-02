package internal_api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Handler handles HTTP requests for internal API
type Handler struct {
	service *Service
}

// NewHandler creates a new internal API handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetConfig handles GET /internal/get-config
// Returns raw JSON config for Xray to consume (not wrapped in CommandResponse)
func (h *Handler) GetConfig(c *gin.Context) {
	log.Info().Str("clientIP", c.ClientIP()).Msg("Internal API: GetConfig called")

	config := h.service.GetConfig()

	if config == nil || len(config) == 0 {
		log.Warn().Msg("Internal API: GetConfig returning empty config")
	} else {
		configKeys := make([]string, 0, len(config))
		for k := range config {
			configKeys = append(configKeys, k)
		}
		log.Info().Strs("configKeys", configKeys).Msg("Internal API: GetConfig returning config")
	}

	c.JSON(http.StatusOK, config)
}
