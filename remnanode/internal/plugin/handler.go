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
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.Sync(&req)
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}

// TorrentBlockerCollect handles POST /node/plugin/torrent-blocker/collect.
func (h *Handler) TorrentBlockerCollect(c *gin.Context) {
	errors.SendSuccess(c, gin.H{"accepted": false})
}

// NftablesBlockIps handles POST /node/plugin/nftables/block-ips.
func (h *Handler) NftablesBlockIps(c *gin.Context) {
	errors.SendSuccess(c, gin.H{"accepted": false})
}

// NftablesUnblockIps handles POST /node/plugin/nftables/unblock-ips.
func (h *Handler) NftablesUnblockIps(c *gin.Context) {
	errors.SendSuccess(c, gin.H{"accepted": false})
}

// NftablesRecreateTables handles POST /node/plugin/nftables/recreate-tables.
func (h *Handler) NftablesRecreateTables(c *gin.Context) {
	errors.SendSuccess(c, gin.H{"accepted": false})
}
