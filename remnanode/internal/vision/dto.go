package vision

// BlockIPRequest represents the request to block an IP
type BlockIPRequest struct {
	IP       string `json:"ip" binding:"required"`
	Username string `json:"username" binding:"required"`
}

// UnblockIPRequest represents the request to unblock an IP
type UnblockIPRequest struct {
	IP       string `json:"ip" binding:"required"`
	Username string `json:"username" binding:"required"`
}
