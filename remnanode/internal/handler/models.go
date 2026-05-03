package handler

// AddUserResponse represents the response for adding a user
type AddUserResponse struct {
	Success bool    `json:"success"`
	Error   *string `json:"error"`
}

// RemoveUserResponse represents the response for removing a user
type RemoveUserResponse struct {
	Success bool    `json:"success"`
	Error   *string `json:"error"`
}

// InboundUser represents a user in an inbound
type InboundUser struct {
	Username string `json:"username"`
	Level    uint32 `json:"level,omitempty"`
	Protocol string `json:"protocol"`
}

// GetInboundUsersResponse represents the response for getting inbound users
type GetInboundUsersResponse struct {
	Users []InboundUser `json:"users"`
}

// GetInboundUsersCountResponse represents the response for getting inbound users count
type GetInboundUsersCountResponse struct {
	Count int `json:"count"`
}

// GenericResponse represents a generic success/failure response
type GenericResponse struct {
	Success bool `json:"success"`
}
