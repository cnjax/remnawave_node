package stats

// GetUserOnlineStatusRequest represents the request for user online status
type GetUserOnlineStatusRequest struct {
	Username string `json:"username" binding:"required"`
}

// GetUsersStatsRequest represents the request for users stats
type GetUsersStatsRequest struct {
	Reset bool `json:"reset"`
}

// GetInboundStatsRequest represents the request for inbound stats
type GetInboundStatsRequest struct {
	Tag   string `json:"tag" binding:"required"`
	Reset bool   `json:"reset"`
}

// GetOutboundStatsRequest represents the request for outbound stats
type GetOutboundStatsRequest struct {
	Tag   string `json:"tag" binding:"required"`
	Reset bool   `json:"reset"`
}

// GetAllInboundsStatsRequest represents the request for all inbounds stats
type GetAllInboundsStatsRequest struct {
	Reset bool `json:"reset"`
}

// GetAllOutboundsStatsRequest represents the request for all outbounds stats
type GetAllOutboundsStatsRequest struct {
	Reset bool `json:"reset"`
}

// GetCombinedStatsRequest represents the request for combined stats
type GetCombinedStatsRequest struct {
	Reset bool `json:"reset"`
}
