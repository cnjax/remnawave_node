package xray

import (
	"bytes"
	"fmt"
	"io"

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
	log.Info().Msg("Handler.Start called")

	// Log content-type and raw body for debugging
	contentType := c.GetHeader("Content-Type")
	log.Info().Str("content-type", contentType).Msg("Request content type")

	// Read the raw body for debugging
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read request body")
		errors.SendErrorWithMessage(c, errors.ErrInternalServer, "Failed to read body")
		return
	}

	// Log first bytes as hex to identify compression format
	if len(bodyBytes) >= 10 {
		log.Info().
			Str("first_10_bytes_hex", fmt.Sprintf("%x", bodyBytes[:10])).
			Int("total_length", len(bodyBytes)).
			Str("content_encoding", c.GetHeader("Content-Encoding")).
			Msg("Request body hex preview")
	}

	// Log first 500 chars of body for debugging
	bodyStr := string(bodyBytes)
	if len(bodyStr) > 500 {
		log.Info().Str("body_preview", bodyStr[:500]).Int("total_length", len(bodyStr)).Msg("Request body preview")
	} else {
		log.Info().Str("body", bodyStr).Msg("Request body")
	}

	// Restore body for binding
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req StartXrayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Str("body_start", bodyStr[:min(100, len(bodyStr))]).Msg("Failed to bind JSON for xray start request")
		errors.SendErrorWithMessage(c, errors.ErrInternalServer, err.Error())
		return
	}

	log.Info().
		Int("inbounds_count", len(req.Internals.Hashes.Inbounds)).
		Bool("force_restart", req.Internals.ForceRestart).
		Str("empty_config_hash", req.Internals.Hashes.EmptyConfig).
		Msg("Received xray start request - JSON parsed successfully")

	// Log inbound tags
	for i, inb := range req.Internals.Hashes.Inbounds {
		log.Info().
			Int("index", i).
			Str("tag", inb.Tag).
			Str("hash", inb.Hash).
			Int("usersCount", inb.UsersCount).
			Msg("Inbound hash info")
	}

	// Log xrayConfig keys
	configKeys := make([]string, 0, len(req.XrayConfig))
	for k := range req.XrayConfig {
		configKeys = append(configKeys, k)
	}
	log.Info().Strs("config_keys", configKeys).Msg("XrayConfig top-level keys")

	clientIP := c.ClientIP()
	log.Info().Str("clientIP", clientIP).Msg("Calling service.StartXray")

	resp, err := h.service.StartXray(&req, clientIP)
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

	log.Info().
		Bool("is_started", resp.IsStarted).
		Msg("Xray start response")

	errors.SendSuccess(c, resp)
}

// Stop handles GET /node/xray/stop
func (h *Handler) Stop(c *gin.Context) {
	resp, err := h.service.StopXray()
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetStatus handles GET /node/xray/status
func (h *Handler) GetStatus(c *gin.Context) {
	resp, err := h.service.GetStatus()
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetNodeHealthCheck handles GET /node/xray/node-health-check
func (h *Handler) GetNodeHealthCheck(c *gin.Context) {
	resp, err := h.service.GetNodeHealthCheck()
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}
