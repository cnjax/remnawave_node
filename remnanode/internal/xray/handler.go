package xray

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/internal/errors"
)

// Handler handles HTTP requests for Xray management
type Handler struct {
	service *Service
}

// NewHandler creates a new Xray handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Start handles POST /node/xray/start
func (h *Handler) Start(c *gin.Context) {
	var req StartXrayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("Failed to bind xray start request")
		errors.SendErrorWithMessage(c, errors.ErrInternalServer, err.Error())
		return
	}

	resp, err := h.service.StartXray(&req, c.ClientIP())
	if err != nil {
		log.Error().Err(err).Msg("StartXray service error")
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	if resp == nil {
		log.Error().Msg("StartXray returned nil response")
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	log.Info().Bool("is_started", resp.IsStarted).Msg("Xray start response")
	errors.SendSuccess(c, resp)
}

// Stop handles GET /node/xray/stop
func (h *Handler) Stop(c *gin.Context) {
	log.Info().Str("clientIP", c.ClientIP()).Msg("Xray stop requested")
	resp, err := h.service.StopXray()
	if err != nil {
		log.Error().Err(err).Msg("StopXray service error")
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	log.Info().Bool("is_stopped", resp.IsStopped).Msg("Xray stop response")
	errors.SendSuccess(c, resp)
}

// GetStatus handles GET /node/xray/status
func (h *Handler) GetStatus(c *gin.Context) {
	log.Info().Str("clientIP", c.ClientIP()).Msg("Xray status requested")
	resp, err := h.service.GetStatus()
	if err != nil {
		log.Error().Err(err).Msg("GetStatus service error")
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	log.Info().Bool("is_running", resp.IsRunning).Str("version", resp.Version).Msg("Xray status response")
	errors.SendSuccess(c, resp)
}

// GetNodeHealthCheck handles GET /node/xray/node-health-check
func (h *Handler) GetNodeHealthCheck(c *gin.Context) {
	log.Info().Str("clientIP", c.ClientIP()).Msg("Xray healthcheck requested")
	resp, err := h.service.GetNodeHealthCheck()
	if err != nil {
		log.Error().Err(err).Msg("GetNodeHealthCheck service error")
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	log.Info().
		Bool("is_alive", resp.IsAlive).
		Bool("xray_internal_status_cached", resp.XrayInternalStatusCached).
		Msg("Xray healthcheck response")
	errors.SendSuccess(c, resp)
}
