package vision

import (
	"github.com/gin-gonic/gin"

	"github.com/remnawave/remnanode/internal/errors"
)

// Handler handles HTTP requests for IP blocking
type Handler struct {
	service *Service
}

// NewHandler creates a new vision handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// BlockIP handles POST /block-ip
func (h *Handler) BlockIP(c *gin.Context) {
	var req BlockIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrInternalServer, err.Error())
		return
	}

	resp, err := h.service.BlockIP(&req)
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}

// UnblockIP handles POST /unblock-ip
func (h *Handler) UnblockIP(c *gin.Context) {
	var req UnblockIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrInternalServer, err.Error())
		return
	}

	resp, err := h.service.UnblockIP(&req)
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}
