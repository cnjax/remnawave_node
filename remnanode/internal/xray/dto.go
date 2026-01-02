package xray

import "github.com/remnawave/remnanode/internal/state"

// StartXrayRequest represents the request to start Xray
type StartXrayRequest struct {
	XrayConfig map[string]interface{} `json:"xrayConfig" binding:"required"`
	Internals  struct {
		ForceRestart bool               `json:"forceRestart"`
		Hashes       state.HashesPayload `json:"hashes" binding:"required"`
	} `json:"internals" binding:"required"`
}
