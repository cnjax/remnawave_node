package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/remnawave/remnanode/internal/errors"
	"github.com/remnawave/remnanode/internal/handler"
	internalapi "github.com/remnawave/remnanode/internal/internal_api"
	"github.com/remnawave/remnanode/internal/plugin"
	"github.com/remnawave/remnanode/internal/server/middleware"
	"github.com/remnawave/remnanode/internal/stats"
	"github.com/remnawave/remnanode/internal/vision"
	"github.com/remnawave/remnanode/internal/xray"
)

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check endpoint (no auth required)
	s.mainRouter.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Main API routes with JWT authentication
	nodeGroup := s.mainRouter.Group("/node")
	nodeGroup.Use(middleware.JWTMiddleware(s.config.JWTPublicKey))

	// Handler routes - user management
	handlerGroup := nodeGroup.Group("/handler")
	s.setupHandlerRoutes(handlerGroup)

	// Stats routes - statistics
	statsGroup := nodeGroup.Group("/stats")
	s.setupStatsRoutes(statsGroup)

	// Xray routes - process management
	xrayGroup := nodeGroup.Group("/xray")
	s.setupXrayRoutes(xrayGroup)

	// Plugin routes - panel plugin synchronization
	pluginGroup := nodeGroup.Group("/plugin")
	s.setupPluginRoutes(pluginGroup)

	// Internal router routes (no JWT, but localhost only)
	s.setupInternalRoutes()
}

// visionGroup is the route group for vision (IP blocking) endpoints
const visionGroupPath = "/vision"

// setupHandlerRoutes configures handler module routes
func (s *Server) setupHandlerRoutes(group *gin.RouterGroup) {
	h, ok := s.services.Handler.(*handler.Handler)
	if !ok || h == nil {
		group.POST("/add-user", notImplemented)
		group.POST("/add-users", notImplemented)
		group.POST("/remove-user", notImplemented)
		group.POST("/remove-users", notImplemented)
		group.POST("/get-inbound-users", notImplemented)
		group.POST("/get-inbound-users-count", notImplemented)
		group.POST("/drop-users-connections", notImplemented)
		group.POST("/drop-ips", notImplemented)
		return
	}

	group.POST("/add-user", h.AddUser)
	group.POST("/add-users", h.AddUsers)
	group.POST("/remove-user", h.RemoveUser)
	group.POST("/remove-users", h.RemoveUsers)
	group.POST("/get-inbound-users", h.GetInboundUsers)
	group.POST("/get-inbound-users-count", h.GetInboundUsersCount)
	group.POST("/drop-users-connections", h.DropUsersConnections)
	group.POST("/drop-ips", h.DropIps)
}

// setupStatsRoutes configures stats module routes
func (s *Server) setupStatsRoutes(group *gin.RouterGroup) {
	st, ok := s.services.Stats.(*stats.Handler)
	if !ok || st == nil {
		group.GET("/get-system-stats", notImplemented)
		group.POST("/get-user-online-status", notImplemented)
		group.POST("/get-users-stats", notImplemented)
		group.POST("/get-inbound-stats", notImplemented)
		group.POST("/get-outbound-stats", notImplemented)
		group.POST("/get-all-inbounds-stats", notImplemented)
		group.POST("/get-all-outbounds-stats", notImplemented)
		group.POST("/get-combined-stats", notImplemented)
		group.POST("/get-user-ip-list", notImplemented)
		group.GET("/get-users-ip-list", notImplemented)
		return
	}

	group.GET("/get-system-stats", st.GetSystemStats)
	group.POST("/get-user-online-status", st.GetUserOnlineStatus)
	group.POST("/get-users-stats", st.GetUsersStats)
	group.POST("/get-inbound-stats", st.GetInboundStats)
	group.POST("/get-outbound-stats", st.GetOutboundStats)
	group.POST("/get-all-inbounds-stats", st.GetAllInboundsStats)
	group.POST("/get-all-outbounds-stats", st.GetAllOutboundsStats)
	group.POST("/get-combined-stats", st.GetCombinedStats)
	group.POST("/get-user-ip-list", st.GetUserIpList)
	group.GET("/get-users-ip-list", st.GetUsersIpList)
}

// setupXrayRoutes configures xray module routes
func (s *Server) setupXrayRoutes(group *gin.RouterGroup) {
	x, ok := s.services.Xray.(*xray.Handler)
	if !ok || x == nil {
		group.POST("/start", notImplemented)
		group.GET("/stop", notImplemented)
		group.GET("/status", notImplemented)
		group.GET("/healthcheck", notImplemented)
		return
	}

	group.POST("/start", x.Start)
	group.GET("/stop", x.Stop)
	group.GET("/status", x.GetStatus)
	group.GET("/healthcheck", x.GetNodeHealthCheck)
}

// setupPluginRoutes configures plugin module routes
func (s *Server) setupPluginRoutes(group *gin.RouterGroup) {
	p, ok := s.services.Plugin.(*plugin.Handler)
	if !ok || p == nil {
		group.POST("/sync", notImplemented)
		group.POST("/torrent-blocker/collect", notImplemented)
		group.POST("/nftables/block-ips", notImplemented)
		group.POST("/nftables/unblock-ips", notImplemented)
		group.POST("/nftables/recreate-tables", notImplemented)
		return
	}

	group.POST("/sync", p.Sync)
	group.POST("/torrent-blocker/collect", p.TorrentBlockerCollect)
	group.POST("/nftables/block-ips", p.NftablesBlockIps)
	group.POST("/nftables/unblock-ips", p.NftablesUnblockIps)
	group.POST("/nftables/recreate-tables", p.NftablesRecreateTables)
}

// setupInternalRoutes configures internal API routes (localhost only)
func (s *Server) setupInternalRoutes() {
	// Vision routes - IP blocking under /vision (matches TS contract VISION_CONTROLLER='vision')
	visionGroup := s.internalRouter.Group(visionGroupPath)
	v, ok := s.services.Vision.(*vision.Handler)
	if ok && v != nil {
		visionGroup.POST("/block-ip", v.BlockIP)
		visionGroup.POST("/unblock-ip", v.UnblockIP)
	} else {
		visionGroup.POST("/block-ip", notImplemented)
		visionGroup.POST("/unblock-ip", notImplemented)
	}

	// Internal API routes
	internal, ok := s.services.InternalAPI.(*internalapi.Handler)
	if ok && internal != nil {
		s.internalRouter.GET("/internal/get-config", internal.GetConfig)
		s.internalRouter.POST("/internal/webhook", internal.Webhook)
	} else {
		s.internalRouter.GET("/internal/get-config", notImplemented)
		s.internalRouter.POST("/internal/webhook", notImplemented)
	}
}

// notImplemented is a placeholder handler
func notImplemented(c *gin.Context) {
	errors.SendError(c, errors.ErrNotImplemented)
}
