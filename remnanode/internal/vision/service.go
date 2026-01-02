package vision

import (
	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/internal/xray_client"
)

// Service handles IP blocking operations
type Service struct {
	xrayClient *xray_client.Client
}

// NewService creates a new vision service
func NewService(xrayClient *xray_client.Client) *Service {
	return &Service{xrayClient: xrayClient}
}

// BlockIP blocks an IP address
func (s *Service) BlockIP(req *BlockIPRequest) (*BlockIPResponse, error) {
	err := s.xrayClient.BlockIP(req.IP, req.Username)
	if err != nil {
		log.Error().
			Err(err).
			Str("ip", req.IP).
			Str("username", req.Username).
			Msg("Failed to block IP")
		return &BlockIPResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	log.Info().
		Str("ip", req.IP).
		Str("username", req.Username).
		Msg("Blocked IP")

	return &BlockIPResponse{
		Success: true,
		Error:   "",
	}, nil
}

// UnblockIP unblocks an IP address
func (s *Service) UnblockIP(req *UnblockIPRequest) (*UnblockIPResponse, error) {
	err := s.xrayClient.UnblockIP(req.IP, req.Username)
	if err != nil {
		log.Error().
			Err(err).
			Str("ip", req.IP).
			Str("username", req.Username).
			Msg("Failed to unblock IP")
		return &UnblockIPResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	log.Info().
		Str("ip", req.IP).
		Str("username", req.Username).
		Msg("Unblocked IP")

	return &UnblockIPResponse{
		Success: true,
		Error:   "",
	}, nil
}
