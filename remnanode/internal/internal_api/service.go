package internal_api

import "github.com/remnawave/remnanode/internal/state"

// Service handles internal API operations
type Service struct {
	stateManager *state.Manager
}

// NewService creates a new internal API service
func NewService(stateManager *state.Manager) *Service {
	return &Service{stateManager: stateManager}
}

// GetConfig returns the current Xray configuration
func (s *Service) GetConfig() map[string]interface{} {
	return s.stateManager.GetXrayConfig()
}
