package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/remnawave/remnanode/internal/errors"
)

// Handler handles HTTP requests for user management
type Handler struct {
	service *Service
}

// NewHandler creates a new handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// AddUser handles POST /node/handler/add-user
func (h *Handler) AddUser(c *gin.Context) {
	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.AddUser(&req)
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}

// AddUsers handles POST /node/handler/add-users
func (h *Handler) AddUsers(c *gin.Context) {
	var req AddUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.AddUsers(&req)
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}

// RemoveUser handles POST /node/handler/remove-user
func (h *Handler) RemoveUser(c *gin.Context) {
	var req RemoveUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.RemoveUser(&req)
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}

// RemoveUsers handles POST /node/handler/remove-users
func (h *Handler) RemoveUsers(c *gin.Context) {
	var req RemoveUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.RemoveUsers(&req)
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetInboundUsers handles POST /node/handler/get-inbound-users
func (h *Handler) GetInboundUsers(c *gin.Context) {
	var req GetInboundUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.GetInboundUsers(req.Tag)
	if err != nil {
		errors.SendError(c, errors.ErrFailedToGetInboundUsers)
		return
	}

	errors.SendSuccess(c, resp)
}

// GetInboundUsersCount handles POST /node/handler/get-inbound-users-count
func (h *Handler) GetInboundUsersCount(c *gin.Context) {
	var req GetInboundUsersCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.GetInboundUsersCount(req.Tag)
	if err != nil {
		errors.SendError(c, errors.ErrFailedToGetInboundUsers)
		return
	}

	errors.SendSuccess(c, resp)
}

// DropUsersConnections handles POST /node/handler/drop-users-connections
func (h *Handler) DropUsersConnections(c *gin.Context) {
	var req DropUsersConnectionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.DropUsersConnections(&req)
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}

// DropIps handles POST /node/handler/drop-ips
func (h *Handler) DropIps(c *gin.Context) {
	var req DropIpsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.SendErrorWithMessage(c, errors.ErrValidation, err.Error())
		return
	}

	resp, err := h.service.DropIps(&req)
	if err != nil {
		errors.SendError(c, errors.ErrInternalServer)
		return
	}

	errors.SendSuccess(c, resp)
}
