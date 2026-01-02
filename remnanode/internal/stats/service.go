package stats

import (
	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/internal/xray_client"
)

// Service handles statistics operations
type Service struct {
	xrayClient *xray_client.Client
}

// NewService creates a new stats service
func NewService(xrayClient *xray_client.Client) *Service {
	return &Service{xrayClient: xrayClient}
}

// GetSystemStats retrieves system statistics from Xray
func (s *Service) GetSystemStats() (*GetSystemStatsResponse, error) {
	stats, err := s.xrayClient.GetSysStats()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get system stats")
		return nil, err
	}

	return &GetSystemStatsResponse{
		NumGoroutine: stats.NumGoroutine,
		NumGC:        stats.NumGC,
		Alloc:        stats.Alloc,
		TotalAlloc:   stats.TotalAlloc,
		Sys:          stats.Sys,
		Mallocs:      stats.Mallocs,
		Frees:        stats.Frees,
		LiveObjects:  stats.LiveObjects,
		PauseTotalNs: stats.PauseTotalNs,
		Uptime:       stats.Uptime,
	}, nil
}

// GetUserOnlineStatus checks if a user is currently online
func (s *Service) GetUserOnlineStatus(username string) (*GetUserOnlineStatusResponse, error) {
	online, err := s.xrayClient.GetUserOnlineStatus(username)
	if err != nil {
		log.Error().Err(err).Str("username", username).Msg("Failed to get user online status")
		return &GetUserOnlineStatusResponse{Online: false}, nil
	}

	return &GetUserOnlineStatusResponse{Online: online}, nil
}

// GetUsersStats retrieves stats for all users
func (s *Service) GetUsersStats(reset bool) (*GetUsersStatsResponse, error) {
	stats, err := s.xrayClient.GetAllUsersStats(reset)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get users stats")
		return nil, err
	}

	// Filter out users with no traffic
	filtered := make([]UserStats, 0)
	for _, u := range stats {
		if u.Uplink != 0 || u.Downlink != 0 {
			filtered = append(filtered, UserStats{
				Username: u.Username,
				Uplink:   u.Uplink,
				Downlink: u.Downlink,
			})
		}
	}

	return &GetUsersStatsResponse{Users: filtered}, nil
}

// GetInboundStats retrieves stats for a specific inbound
func (s *Service) GetInboundStats(tag string, reset bool) (*GetInboundStatsResponse, error) {
	stats, err := s.xrayClient.GetInboundStats(tag, reset)
	if err != nil {
		return nil, err
	}

	return &GetInboundStatsResponse{
		Inbound:  stats.Inbound,
		Uplink:   stats.Uplink,
		Downlink: stats.Downlink,
	}, nil
}

// GetOutboundStats retrieves stats for a specific outbound
func (s *Service) GetOutboundStats(tag string, reset bool) (*GetOutboundStatsResponse, error) {
	stats, err := s.xrayClient.GetOutboundStats(tag, reset)
	if err != nil {
		return nil, err
	}

	return &GetOutboundStatsResponse{
		Outbound: stats.Outbound,
		Uplink:   stats.Uplink,
		Downlink: stats.Downlink,
	}, nil
}

// GetAllInboundsStats retrieves stats for all inbounds
func (s *Service) GetAllInboundsStats(reset bool) (*GetAllInboundsStatsResponse, error) {
	stats, err := s.xrayClient.GetAllInboundsStats(reset)
	if err != nil {
		return nil, err
	}

	return &GetAllInboundsStatsResponse{Inbounds: stats}, nil
}

// GetAllOutboundsStats retrieves stats for all outbounds
func (s *Service) GetAllOutboundsStats(reset bool) (*GetAllOutboundsStatsResponse, error) {
	stats, err := s.xrayClient.GetAllOutboundsStats(reset)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get all outbounds stats")
		return nil, err
	}

	return &GetAllOutboundsStatsResponse{Outbounds: stats}, nil
}

// GetCombinedStats retrieves combined stats for inbounds and outbounds
func (s *Service) GetCombinedStats(reset bool) (*GetCombinedStatsResponse, error) {
	inbounds, err := s.xrayClient.GetAllInboundsStats(reset)
	if err != nil {
		return nil, err
	}

	outbounds, err := s.xrayClient.GetAllOutboundsStats(reset)
	if err != nil {
		return nil, err
	}

	return &GetCombinedStatsResponse{
		Inbounds:  inbounds,
		Outbounds: outbounds,
	}, nil
}
