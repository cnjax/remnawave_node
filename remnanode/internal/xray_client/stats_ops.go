package xray_client

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	statsService "github.com/xtls/xray-core/app/stats/command"
)

// SysStats represents system statistics from Xray
type SysStats struct {
	NumGoroutine uint32 `json:"numGoroutine"`
	NumGC        uint32 `json:"numGC"`
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"totalAlloc"`
	Sys          uint64 `json:"sys"`
	Mallocs      uint64 `json:"mallocs"`
	Frees        uint64 `json:"frees"`
	LiveObjects  uint64 `json:"liveObjects"`
	PauseTotalNs uint64 `json:"pauseTotalNs"`
	Uptime       uint32 `json:"uptime"`
}

// UserStats represents user traffic statistics
type UserStats struct {
	Username string `json:"username"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

// InboundStats represents inbound traffic statistics
type InboundStats struct {
	Inbound  string `json:"inbound"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

// OutboundStats represents outbound traffic statistics
type OutboundStats struct {
	Outbound string `json:"outbound"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

// GetSysStats gets system statistics from Xray
func (c *Client) GetSysStats() (*SysStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.stats.GetSysStats(ctx, &statsService.SysStatsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get sys stats: %w", err)
	}

	return &SysStats{
		NumGoroutine: resp.NumGoroutine,
		NumGC:        resp.NumGC,
		Alloc:        resp.Alloc,
		TotalAlloc:   resp.TotalAlloc,
		Sys:          resp.Sys,
		Mallocs:      resp.Mallocs,
		Frees:        resp.Frees,
		LiveObjects:  resp.LiveObjects,
		PauseTotalNs: resp.PauseTotalNs,
		Uptime:       resp.Uptime,
	}, nil
}

// GetUserOnlineStatus checks if a user is currently online via the GetStats
// online counter (user>>>{name}>>>online), matching TS xtls-sdk getUserOnlineStatus.
// Returns true only when the counter value is > 0.
func (c *Client) GetUserOnlineStatus(username string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.stats.GetStats(ctx, &statsService.GetStatsRequest{
		Name:   fmt.Sprintf("user>>>%s>>>online", username),
		Reset_: false,
	})
	if err != nil {
		// xray returns not-found when the user has no online counter — treat as offline.
		return false, nil
	}

	return resp.Stat != nil && resp.Stat.Value > 0, nil
}

// GetAllUsersStats gets statistics for all users
func (c *Client) GetAllUsersStats(reset bool) ([]UserStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.stats.QueryStats(ctx, &statsService.QueryStatsRequest{
		Pattern: "user>>>",
		Reset_:  reset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query all users stats: %w", err)
	}

	// Parse stats into user stats map
	userStatsMap := make(map[string]*UserStats)
	userPattern := regexp.MustCompile(`user>>>(.+?)>>>traffic>>>(uplink|downlink)`)

	for _, stat := range resp.Stat {
		matches := userPattern.FindStringSubmatch(stat.Name)
		if len(matches) != 3 {
			continue
		}

		username := matches[1]
		direction := matches[2]

		if _, exists := userStatsMap[username]; !exists {
			userStatsMap[username] = &UserStats{Username: username}
		}

		if direction == "uplink" {
			userStatsMap[username].Uplink = stat.Value
		} else if direction == "downlink" {
			userStatsMap[username].Downlink = stat.Value
		}
	}

	// Convert map to slice
	users := make([]UserStats, 0, len(userStatsMap))
	for _, u := range userStatsMap {
		users = append(users, *u)
	}

	return users, nil
}

// GetInboundStats gets statistics for a specific inbound
func (c *Client) GetInboundStats(tag string, reset bool) (*InboundStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.stats.QueryStats(ctx, &statsService.QueryStatsRequest{
		Pattern: fmt.Sprintf("inbound>>>%s>>>traffic>>>", tag),
		Reset_:  reset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query inbound stats: %w", err)
	}

	stats := &InboundStats{Inbound: tag}
	for _, stat := range resp.Stat {
		if strings.HasSuffix(stat.Name, "uplink") {
			stats.Uplink = stat.Value
		} else if strings.HasSuffix(stat.Name, "downlink") {
			stats.Downlink = stat.Value
		}
	}

	return stats, nil
}

// GetOutboundStats gets statistics for a specific outbound
func (c *Client) GetOutboundStats(tag string, reset bool) (*OutboundStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.stats.QueryStats(ctx, &statsService.QueryStatsRequest{
		Pattern: fmt.Sprintf("outbound>>>%s>>>traffic>>>", tag),
		Reset_:  reset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query outbound stats: %w", err)
	}

	stats := &OutboundStats{Outbound: tag}
	for _, stat := range resp.Stat {
		if strings.HasSuffix(stat.Name, "uplink") {
			stats.Uplink = stat.Value
		} else if strings.HasSuffix(stat.Name, "downlink") {
			stats.Downlink = stat.Value
		}
	}

	return stats, nil
}

// GetAllInboundsStats gets statistics for all inbounds
func (c *Client) GetAllInboundsStats(reset bool) ([]InboundStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.stats.QueryStats(ctx, &statsService.QueryStatsRequest{
		Pattern: "inbound>>>",
		Reset_:  reset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query all inbounds stats: %w", err)
	}

	// Parse stats into inbound stats map
	inboundStatsMap := make(map[string]*InboundStats)
	inboundPattern := regexp.MustCompile(`inbound>>>(.+?)>>>traffic>>>(uplink|downlink)`)

	for _, stat := range resp.Stat {
		matches := inboundPattern.FindStringSubmatch(stat.Name)
		if len(matches) != 3 {
			continue
		}

		tag := matches[1]
		direction := matches[2]

		if _, exists := inboundStatsMap[tag]; !exists {
			inboundStatsMap[tag] = &InboundStats{Inbound: tag}
		}

		if direction == "uplink" {
			inboundStatsMap[tag].Uplink = stat.Value
		} else if direction == "downlink" {
			inboundStatsMap[tag].Downlink = stat.Value
		}
	}

	// Convert map to slice
	inbounds := make([]InboundStats, 0, len(inboundStatsMap))
	for _, i := range inboundStatsMap {
		inbounds = append(inbounds, *i)
	}

	return inbounds, nil
}

// GetAllOutboundsStats gets statistics for all outbounds
func (c *Client) GetAllOutboundsStats(reset bool) ([]OutboundStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.stats.QueryStats(ctx, &statsService.QueryStatsRequest{
		Pattern: "outbound>>>",
		Reset_:  reset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query all outbounds stats: %w", err)
	}

	// Parse stats into outbound stats map
	outboundStatsMap := make(map[string]*OutboundStats)
	outboundPattern := regexp.MustCompile(`outbound>>>(.+?)>>>traffic>>>(uplink|downlink)`)

	for _, stat := range resp.Stat {
		matches := outboundPattern.FindStringSubmatch(stat.Name)
		if len(matches) != 3 {
			continue
		}

		tag := matches[1]
		direction := matches[2]

		if _, exists := outboundStatsMap[tag]; !exists {
			outboundStatsMap[tag] = &OutboundStats{Outbound: tag}
		}

		if direction == "uplink" {
			outboundStatsMap[tag].Uplink = stat.Value
		} else if direction == "downlink" {
			outboundStatsMap[tag].Downlink = stat.Value
		}
	}

	// Convert map to slice
	outbounds := make([]OutboundStats, 0, len(outboundStatsMap))
	for _, o := range outboundStatsMap {
		outbounds = append(outbounds, *o)
	}

	return outbounds, nil
}

// OnlineUserIp represents an IP entry with last-seen timestamp
type OnlineUserIp struct {
	IP       string
	LastSeen int64 // unix timestamp
}

// GetStatsOnlineIpList gets the list of IPs for a user (name = "user>>>userId>>>online").
// Pass reset=true only when the caller intentionally wants to clear the counter after reading.
func (c *Client) GetStatsOnlineIpList(name string, reset bool) ([]OnlineUserIp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.stats.GetStatsOnlineIpList(ctx, &statsService.GetStatsRequest{Name: name, Reset_: reset})
	if err != nil {
		return nil, err
	}

	result := make([]OnlineUserIp, 0, len(resp.Ips))
	for ip, ts := range resp.Ips {
		result = append(result, OnlineUserIp{IP: ip, LastSeen: ts})
	}
	return result, nil
}

// GetAllOnlineUsers returns all online user stat names
func (c *Client) GetAllOnlineUsers() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.stats.GetAllOnlineUsers(ctx, &statsService.GetAllOnlineUsersRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Users, nil
}

