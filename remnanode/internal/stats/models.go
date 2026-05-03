package stats

import (
	"time"

	"github.com/remnawave/remnanode/internal/xray_client"
	"github.com/remnawave/remnanode/pkg/sysinfo"
)

// XrayInfo holds xray process statistics
type XrayInfo struct {
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

// PluginStats matches the current TS plugin stats response shape.
type PluginStats struct {
	TorrentBlocker TorrentBlockerStats `json:"torrentBlocker"`
}

type TorrentBlockerStats struct {
	ReportsCount int `json:"reportsCount"`
}

type SystemStatsContainer struct {
	Stats *sysinfo.SystemStats `json:"stats"`
}

// GetSystemStatsResponse matches the current upstream TS contract:
// { xrayInfo, plugins, system: { stats } }.
type GetSystemStatsResponse struct {
	XrayInfo *XrayInfo            `json:"xrayInfo"`
	Plugins  PluginStats          `json:"plugins"`
	System   SystemStatsContainer `json:"system"`
}

// GetUserOnlineStatusResponse represents the response for user online status.
// Field name matches the TS contract (libs/contract/commands/stats/get-user-online-status.command.ts).
type GetUserOnlineStatusResponse struct {
	IsOnline bool `json:"isOnline"`
}

// UserStats represents a single user's stats
type UserStats struct {
	Username string `json:"username"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

// GetUsersStatsResponse represents the response for all users stats
type GetUsersStatsResponse struct {
	Users []UserStats `json:"users"`
}

// GetInboundStatsResponse represents the response for inbound stats
type GetInboundStatsResponse struct {
	Inbound  string `json:"inbound"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

// GetOutboundStatsResponse represents the response for outbound stats
type GetOutboundStatsResponse struct {
	Outbound string `json:"outbound"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

// GetAllInboundsStatsResponse represents the response for all inbounds stats
type GetAllInboundsStatsResponse struct {
	Inbounds []xray_client.InboundStats `json:"inbounds"`
}

// GetAllOutboundsStatsResponse represents the response for all outbounds stats
type GetAllOutboundsStatsResponse struct {
	Outbounds []xray_client.OutboundStats `json:"outbounds"`
}

// GetCombinedStatsResponse represents the response for combined stats
type GetCombinedStatsResponse struct {
	Inbounds  []xray_client.InboundStats  `json:"inbounds"`
	Outbounds []xray_client.OutboundStats `json:"outbounds"`
}

// DetailedIP represents an IP with last-seen timestamp
type DetailedIP struct {
	IP       string    `json:"ip"`
	LastSeen time.Time `json:"lastSeen"`
}

// GetUserIpListResponse represents the response for user IP list
type GetUserIpListResponse struct {
	IPs []DetailedIP `json:"ips"`
}

// UserIpList holds IPs for a single user
type UserIpList struct {
	UserID string       `json:"userId"`
	IPs    []DetailedIP `json:"ips"`
}

// GetUsersIpListResponse represents the response for all users IP list
type GetUsersIpListResponse struct {
	Users []UserIpList `json:"users"`
}
