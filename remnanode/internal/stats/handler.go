package stats

import (
	"github.com/gin-gonic/gin"

	"github.com/remnawave/remnanode/internal/errors"
)

// Handler handles HTTP requests for statistics
type Handler struct {
	service *Service
}

// NewHandler creates a new stats handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetSystemStats handles GET /node/stats/system-stats
func (h *Handler) GetSystemStats(c *gin.Context) {
	resp, err := h.service.GetSystemStats()
	if err != nil {
		errors.SendError(c, errors.ErrFailedToGetSystemStats)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetUserOnlineStatus handles POST /node/stats/user-online-status
func (h *Handler) GetUserOnlineStatus(c *gin.Context) {
	var req GetUserOnlineStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.GetUserOnlineStatus(req.Username)
	if err != nil {
		errors.SendSuccess(c, &GetUserOnlineStatusResponse{IsOnline: false})
		return
	}

	errors.SendSuccess(c, resp)
}

// GetUsersStats handles POST /node/stats/get-users-stats
func (h *Handler) GetUsersStats(c *gin.Context) {
	var req GetUsersStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.GetUsersStats(req.Reset)
	if err != nil {
		errors.SendError(c, errors.ErrFailedToGetUsersStats)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetInboundStats handles POST /node/stats/get-inbound-stats
func (h *Handler) GetInboundStats(c *gin.Context) {
	var req GetInboundStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.GetInboundStats(req.Tag, req.Reset)
	if err != nil {
		errors.SendError(c, errors.ErrFailedToGetInboundStats)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetOutboundStats handles POST /node/stats/get-outbound-stats
func (h *Handler) GetOutboundStats(c *gin.Context) {
	var req GetOutboundStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.GetOutboundStats(req.Tag, req.Reset)
	if err != nil {
		errors.SendError(c, errors.ErrFailedToGetOutboundStats)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetAllInboundsStats handles POST /node/stats/get-all-inbounds-stats
func (h *Handler) GetAllInboundsStats(c *gin.Context) {
	var req GetAllInboundsStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.GetAllInboundsStats(req.Reset)
	if err != nil {
		errors.SendError(c, errors.ErrFailedToGetInboundsStats)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetAllOutboundsStats handles POST /node/stats/get-all-outbounds-stats
func (h *Handler) GetAllOutboundsStats(c *gin.Context) {
	var req GetAllOutboundsStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.GetAllOutboundsStats(req.Reset)
	if err != nil {
		errors.SendError(c, errors.ErrFailedToGetOutboundsStats)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetCombinedStats handles POST /node/stats/get-combined-stats
func (h *Handler) GetCombinedStats(c *gin.Context) {
	var req GetCombinedStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.GetCombinedStats(req.Reset)
	if err != nil {
		errors.SendError(c, errors.ErrFailedToGetCombinedStats)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetUserIpList handles POST /node/stats/get-user-ip-list
func (h *Handler) GetUserIpList(c *gin.Context) {
	var req GetUserIpListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.GetUserIpList(req.UserID)
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetUsersIpList handles GET /node/stats/get-users-ip-list
func (h *Handler) GetUsersIpList(c *gin.Context) {
	resp, err := h.service.GetUsersIpList()
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}
