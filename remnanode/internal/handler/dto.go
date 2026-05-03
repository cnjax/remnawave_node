package handler

import "github.com/remnawave/remnanode/internal/xray_client"

// UserType represents the type of user (vless, trojan, shadowsocks)
type UserType string

const (
	UserTypeVless          UserType = "vless"
	UserTypeTrojan         UserType = "trojan"
	UserTypeShadowsocks    UserType = "shadowsocks"
	UserTypeShadowsocks22  UserType = "shadowsocks22"
	UserTypeHysteria       UserType = "hysteria"
)

// InboundUserData represents data for a single user in an inbound
type InboundUserData struct {
	Type       UserType                `json:"type" binding:"required"`
	Tag        string                  `json:"tag" binding:"required"`
	Username   string                  `json:"username" binding:"required"`
	UUID       string                  `json:"uuid,omitempty"`
	Password   string                  `json:"password,omitempty"`
	Flow       string                  `json:"flow,omitempty"`
	CipherType xray_client.CipherType  `json:"cipherType,omitempty"`
	IVCheck    bool                    `json:"ivCheck,omitempty"`
}

// HashData represents hash information for user tracking
type HashData struct {
	VlessUUID     string `json:"vlessUuid" binding:"required"`
	PrevVlessUUID string `json:"prevVlessUuid,omitempty"`
}

// AddUserRequest represents the request to add a single user
type AddUserRequest struct {
	Data     []InboundUserData `json:"data" binding:"required"`
	HashData HashData          `json:"hashData" binding:"required"`
}

// RemoveUserRequest represents the request to remove a single user
type RemoveUserRequest struct {
	Username string   `json:"username" binding:"required"`
	HashData HashData `json:"hashData" binding:"required"`
}

// BulkUserData represents user data for bulk operations
type BulkUserData struct {
	UserID         string `json:"userId" binding:"required"`
	VlessUUID      string `json:"vlessUuid" binding:"required"`
	TrojanPassword string `json:"trojanPassword,omitempty"`
	SSPassword     string `json:"ssPassword,omitempty"`
	HashUUID       string `json:"hashUuid" binding:"required"`
}

// BulkInboundConfig represents inbound configuration for bulk operations
type BulkInboundConfig struct {
	Type       UserType `json:"type" binding:"required"`
	Tag        string   `json:"tag" binding:"required"`
	Flow       string   `json:"flow,omitempty"`
}

// BulkUser represents a user for bulk operations
type BulkUser struct {
	UserData    BulkUserData        `json:"userData" binding:"required"`
	InboundData []BulkInboundConfig `json:"inboundData" binding:"required"`
}

// AddUsersRequest represents the request to add multiple users
type AddUsersRequest struct {
	AffectedInboundTags []string   `json:"affectedInboundTags"` // optional: empty means "sync all known inbounds"
	Users               []BulkUser `json:"users" binding:"required"`
}

// RemoveUserBulkData represents user data for bulk removal
type RemoveUserBulkData struct {
	UserID   string `json:"userId" binding:"required"`
	HashUUID string `json:"hashUuid" binding:"required"`
}

// RemoveUsersRequest represents the request to remove multiple users
type RemoveUsersRequest struct {
	Users []RemoveUserBulkData `json:"users" binding:"required"`
}

// GetInboundUsersRequest represents the request to get inbound users
type GetInboundUsersRequest struct {
	Tag string `json:"tag" binding:"required"`
}

// GetInboundUsersCountRequest represents the request to get inbound users count
type GetInboundUsersCountRequest struct {
	Tag string `json:"tag" binding:"required"`
}

// DropUsersConnectionsRequest represents the request to drop connections for specific users
type DropUsersConnectionsRequest struct {
	UserIDs []string `json:"userIds" binding:"required,min=1"`
}

// DropIpsRequest represents the request to drop connections for specific IPs
type DropIpsRequest struct {
	IPs []string `json:"ips" binding:"required,min=1"`
}
