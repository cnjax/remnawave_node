package stats

// GetUserOnlineStatusRequest represents the request for user online status
type GetUserOnlineStatusRequest struct {
	Username string `json:"username" binding:"required"`
}

// GetInboundStatsRequest represents the request for inbound stats
type GetInboundStatsRequest struct {
	Tag string `json:"tag" binding:"required"`
}

// GetOutboundStatsRequest represents the request for outbound stats
type GetOutboundStatsRequest struct {
	Tag string `json:"tag" binding:"required"`
}
