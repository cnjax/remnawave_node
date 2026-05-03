package plugin

import (
	"github.com/gin-gonic/gin"

	"github.com/remnawave/remnanode/internal/errors"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Sync handles POST /node/plugin/sync.
func (h *Handler) Sync(c *gin.Context) {
	var req SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrInternalServer, err.Error())
		return
	}

	resp, err := h.service.Sync(&req)
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}
