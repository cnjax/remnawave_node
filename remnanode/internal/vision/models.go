package vision

// BlockIPResponse represents the response for blocking an IP
type BlockIPResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// UnblockIPResponse represents the response for unblocking an IP
type UnblockIPResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}
