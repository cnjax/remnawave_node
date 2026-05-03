package stats

import (
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/internal/xray_client"
	"github.com/remnawave/remnanode/pkg/sysinfo"
)

// Service handles statistics operations
type Service struct {
	xrayClient *xray_client.Client
}

// NewService creates a new stats service
func NewService(xrayClient *xray_client.Client) *Service {
	return &Service{xrayClient: xrayClient}
}

// GetSystemStats retrieves system statistics
func (s *Service) GetSystemStats() (*GetSystemStatsResponse, error) {
	rawStats, err := s.xrayClient.GetSysStats()

	var xrayInfo *XrayInfo
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get xray sys stats")
	} else {
		xrayInfo = &XrayInfo{
			NumGoroutine: rawStats.NumGoroutine,
			NumGC:        rawStats.NumGC,
			Alloc:        rawStats.Alloc,
			TotalAlloc:   rawStats.TotalAlloc,
			Sys:          rawStats.Sys,
			Mallocs:      rawStats.Mallocs,
			Frees:        rawStats.Frees,
			LiveObjects:  rawStats.LiveObjects,
			PauseTotalNs: rawStats.PauseTotalNs,
			Uptime:       rawStats.Uptime,
		}
	}

	sysStat, sysErr := sysinfo.GetSystemStats()
	if sysErr != nil {
		log.Warn().Err(sysErr).Msg("Failed to get system stats")
		sysStat = &sysinfo.SystemStats{LoadAvg: []float64{0, 0, 0}}
	}

	var plugins PluginStats
	plugins.TorrentBlocker.ReportsCount = 0

	return &GetSystemStatsResponse{
		XrayInfo: xrayInfo,
		Plugins:  plugins,
		System:   SystemStatsWrapper{Stats: sysStat},
	}, nil
}

// GetUserOnlineStatus checks if a user is currently online
func (s *Service) GetUserOnlineStatus(username string) (*GetUserOnlineStatusResponse, error) {
	online, err := s.xrayClient.GetUserOnlineStatus(username)
	if err != nil {
		log.Error().Err(err).Str("username", username).Msg("Failed to get user online status")
		return &GetUserOnlineStatusResponse{IsOnline: false}, nil
	}

	return &GetUserOnlineStatusResponse{IsOnline: online}, nil
}

// GetUsersStats retrieves stats for all users
func (s *Service) GetUsersStats(reset bool) (*GetUsersStatsResponse, error) {
	stats, err := s.xrayClient.GetAllUsersStats(reset)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get users stats")
		return nil, err
	}

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

// GetUserIpList retrieves the list of IPs for a specific user
func (s *Service) GetUserIpList(userID string) (*GetUserIpListResponse, error) {
	ips, err := s.xrayClient.GetStatsOnlineIpList("user>>>"+userID+">>>online", false)
	if err != nil {
		log.Warn().Err(err).Str("userId", userID).Msg("Failed to get user IP list")
		return &GetUserIpListResponse{IPs: []DetailedIP{}}, nil
	}

	result := make([]DetailedIP, 0, len(ips))
	for _, ip := range ips {
		result = append(result, DetailedIP{
			IP:       ip.IP,
			LastSeen: time.Unix(ip.LastSeen, 0),
		})
	}

	return &GetUserIpListResponse{IPs: result}, nil
}

// GetUsersIpList retrieves IP lists for all online users
func (s *Service) GetUsersIpList() (*GetUsersIpListResponse, error) {
	onlineUsers, err := s.xrayClient.GetAllOnlineUsers()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get all online users")
		return &GetUsersIpListResponse{Users: []UserIpList{}}, nil
	}

	seen := make(map[string]struct{})
	users := make([]UserIpList, 0)

	for _, raw := range onlineUsers {
		// raw format: "user>>>userId>>>online"
		parts := strings.Split(raw, ">>>")
		if len(parts) < 2 {
			continue
		}
		userID := parts[1]
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}

		ips, err := s.xrayClient.GetStatsOnlineIpList("user>>>"+userID+">>>online", false)
		if err != nil {
			continue
		}
		if len(ips) == 0 {
			continue
		}

		details := make([]DetailedIP, 0, len(ips))
		for _, ip := range ips {
			details = append(details, DetailedIP{
				IP:       ip.IP,
				LastSeen: time.Unix(ip.LastSeen, 0),
			})
		}

		users = append(users, UserIpList{
			UserID: userID,
			IPs:    details,
		})
	}

	return &GetUsersIpListResponse{Users: users}, nil
}
