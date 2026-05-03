package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/internal/state"
	"github.com/remnawave/remnanode/internal/xray_client"
	"github.com/remnawave/remnanode/pkg/connkill"
)

// Service handles user management operations
type Service struct {
	xrayClient            *xray_client.Client
	stateManager          *state.Manager
	disableHashedSetCheck bool
}

// NewService creates a new handler service
func NewService(xrayClient *xray_client.Client, stateManager *state.Manager, disableHashedSetCheck bool) *Service {
	return &Service{
		xrayClient:            xrayClient,
		stateManager:          stateManager,
		disableHashedSetCheck: disableHashedSetCheck,
	}
}

// strPtr is a convenience helper that returns a pointer to s.
func strPtr(s string) *string { return &s }

// validateInboundUser checks that type-specific required fields are present.
func validateInboundUser(item InboundUserData) error {
	switch item.Type {
	case UserTypeVless:
		if item.UUID == "" {
			return fmt.Errorf("vless user %q: uuid is required", item.Username)
		}
	case UserTypeTrojan:
		if item.Password == "" {
			return fmt.Errorf("trojan user %q: password is required", item.Username)
		}
	case UserTypeShadowsocks:
		if item.Password == "" || item.CipherType == xray_client.CipherTypeUnknown {
			return fmt.Errorf("shadowsocks user %q: password and cipherType are required", item.Username)
		}
	case UserTypeShadowsocks22:
		if item.Password == "" {
			return fmt.Errorf("shadowsocks22 user %q: password is required", item.Username)
		}
	case UserTypeHysteria:
		if item.Password == "" {
			return fmt.Errorf("hysteria user %q: password is required", item.Username)
		}
	}
	return nil
}

// AddUser adds a single user to the configured inbounds
func (s *Service) AddUser(req *AddUserRequest) (*AddUserResponse, error) {
	// Per-type validation
	for _, item := range req.Data {
		if err := validateInboundUser(item); err != nil {
			return &AddUserResponse{Success: false, Error: strPtr(err.Error())}, nil
		}
	}

	// Add tags to the state manager
	for _, item := range req.Data {
		s.stateManager.AddXtlsConfigInbound(item.Tag)
	}

	// Remove user from all inbounds first
	for _, tag := range s.stateManager.GetXtlsConfigInbounds() {
		log.Debug().
			Str("username", req.Data[0].Username).
			Str("tag", tag).
			Msg("Removing user")

		s.xrayClient.RemoveUser(tag, req.Data[0].Username)

		if !s.disableHashedSetCheck {
			if req.HashData.PrevVlessUUID != "" {
				s.stateManager.RemoveUserFromInbound(tag, req.HashData.PrevVlessUUID)
			} else {
				s.stateManager.RemoveUserFromInbound(tag, req.HashData.VlessUUID)
			}
		}
	}

	// Add user to each inbound
	var lastError error
	successCount := 0

	for _, item := range req.Data {
		log.Debug().
			Str("username", item.Username).
			Str("type", string(item.Type)).
			Msg("Adding user")

		var err error
		switch item.Type {
		case UserTypeVless:
			err = s.xrayClient.AddVlessUser(item.Tag, item.Username, item.UUID, item.Flow)
		case UserTypeTrojan:
			err = s.xrayClient.AddTrojanUser(item.Tag, item.Username, item.Password)
		case UserTypeShadowsocks:
			err = s.xrayClient.AddShadowsocksUser(item.Tag, item.Username, item.Password, item.CipherType, item.IVCheck)
		case UserTypeShadowsocks22:
			err = s.xrayClient.AddShadowsocks2022User(item.Tag, item.Username, item.Password)
		case UserTypeHysteria:
			err = s.xrayClient.AddHysteriaUser(item.Tag, item.Username, item.Password)
		}

		if err != nil {
			lastError = err
			log.Error().Err(err).Str("tag", item.Tag).Msg("Failed to add user")
		} else {
			successCount++
			if !s.disableHashedSetCheck {
				s.stateManager.AddUserToInbound(item.Tag, req.HashData.VlessUUID)
			}
		}
	}

	if successCount == 0 && lastError != nil {
		log.Error().Err(lastError).Msg("Error adding users")
		return &AddUserResponse{Success: false, Error: strPtr(lastError.Error())}, nil
	}

	return &AddUserResponse{Success: true, Error: nil}, nil
}

// RemoveUser removes a single user from all inbounds
func (s *Service) RemoveUser(req *RemoveUserRequest) (*RemoveUserResponse, error) {
	inboundTags := s.stateManager.GetXtlsConfigInbounds()

	if len(inboundTags) == 0 {
		return &RemoveUserResponse{Success: true, Error: nil}, nil
	}

	var lastError error
	successCount := 0

	for _, tag := range inboundTags {
		log.Debug().
			Str("username", req.Username).
			Str("tag", tag).
			Msg("Removing user")

		err := s.xrayClient.RemoveUser(tag, req.Username)
		if !s.disableHashedSetCheck {
			s.stateManager.RemoveUserFromInbound(tag, req.HashData.VlessUUID)
		}

		if err != nil {
			lastError = err
		} else {
			successCount++
		}
	}

	if successCount == 0 && lastError != nil {
		log.Error().Err(lastError).Msg("Error removing user")
		return &RemoveUserResponse{Success: false, Error: strPtr(lastError.Error())}, nil
	}

	return &RemoveUserResponse{Success: true, Error: nil}, nil
}

// AddUsers adds multiple users in bulk
func (s *Service) AddUsers(req *AddUsersRequest) (*AddUserResponse, error) {
	start := time.Now()
	defer func() {
		log.Info().
			Dur("duration", time.Since(start)).
			Msg("Users addition completed")
	}()

	// Add affected inbound tags
	for _, tag := range req.AffectedInboundTags {
		s.stateManager.AddXtlsConfigInbound(tag)
	}

	log.Info().
		Int("users", len(req.Users)).
		Strs("inbounds", req.AffectedInboundTags).
		Msg("Adding users to inbounds")

	successCount := 0
	failures := make([]string, 0)

	for _, user := range req.Users {
		// Remove user from all inbounds first
		for _, tag := range s.stateManager.GetXtlsConfigInbounds() {
			s.xrayClient.RemoveUser(tag, user.UserData.UserID)
			if !s.disableHashedSetCheck {
				s.stateManager.RemoveUserFromInbound(tag, user.UserData.HashUUID)
			}
		}

		// Add user to each inbound
		for _, inbound := range user.InboundData {
			var err error
			switch inbound.Type {
			case UserTypeVless:
				err = s.xrayClient.AddVlessUser(inbound.Tag, user.UserData.UserID, user.UserData.VlessUUID, inbound.Flow)
			case UserTypeTrojan:
				err = s.xrayClient.AddTrojanUser(inbound.Tag, user.UserData.UserID, user.UserData.TrojanPassword)
			case UserTypeShadowsocks:
				err = s.xrayClient.AddShadowsocksUser(inbound.Tag, user.UserData.UserID, user.UserData.SSPassword, xray_client.CipherTypeCHACHA20POLY1305, false)
			case UserTypeShadowsocks22:
				err = s.xrayClient.AddShadowsocks2022User(inbound.Tag, user.UserData.UserID, user.UserData.SSPassword)
			case UserTypeHysteria:
				err = s.xrayClient.AddHysteriaUser(inbound.Tag, user.UserData.UserID, user.UserData.TrojanPassword)
			}

			if err == nil && !s.disableHashedSetCheck {
				s.stateManager.AddUserToInbound(inbound.Tag, user.UserData.VlessUUID)
			}
			if err != nil {
				msg := fmt.Sprintf("user=%s tag=%s type=%s: %v", user.UserData.UserID, inbound.Tag, inbound.Type, err)
				failures = append(failures, msg)
				log.Warn().Err(err).
					Str("userId", user.UserData.UserID).
					Str("tag", inbound.Tag).
					Str("type", string(inbound.Type)).
					Msg("Failed to add user to inbound")
			} else {
				successCount++
			}
		}
	}

	if successCount == 0 && len(failures) > 0 {
		return &AddUserResponse{Success: false, Error: strPtr(strings.Join(failures, "; "))}, nil
	}

	return &AddUserResponse{Success: true, Error: nil}, nil
}

// RemoveUsers removes multiple users in bulk
func (s *Service) RemoveUsers(req *RemoveUsersRequest) (*RemoveUserResponse, error) {
	start := time.Now()
	defer func() {
		log.Info().
			Dur("duration", time.Since(start)).
			Msg("Users removal completed")
	}()

	inboundTags := s.stateManager.GetXtlsConfigInbounds()

	if len(inboundTags) == 0 {
		return &RemoveUserResponse{Success: true, Error: nil}, nil
	}

	log.Info().
		Int("users", len(req.Users)).
		Strs("inbounds", inboundTags).
		Msg("Removing users from inbounds")

	for _, user := range req.Users {
		for _, tag := range inboundTags {
			log.Debug().
				Str("userId", user.UserID).
				Str("tag", tag).
				Msg("Removing user")

			s.xrayClient.RemoveUser(tag, user.UserID)
			if !s.disableHashedSetCheck {
				s.stateManager.RemoveUserFromInbound(tag, user.HashUUID)
			}
		}
	}

	return &RemoveUserResponse{Success: true, Error: nil}, nil
}

// GetInboundUsers gets all users in an inbound
func (s *Service) GetInboundUsers(tag string) (*GetInboundUsersResponse, error) {
	users, err := s.xrayClient.GetInboundUsers(tag)
	if err != nil {
		return &GetInboundUsersResponse{Users: []InboundUser{}}, err
	}

	result := make([]InboundUser, len(users))
	for i, u := range users {
		result[i] = InboundUser{
			Username: u.Username,
			Level:    u.Level,
			Protocol: u.Protocol,
		}
	}

	return &GetInboundUsersResponse{Users: result}, nil
}

// GetInboundUsersCount gets the count of users in an inbound
func (s *Service) GetInboundUsersCount(tag string) (*GetInboundUsersCountResponse, error) {
	count, err := s.xrayClient.GetInboundUsersCount(tag)
	if err != nil {
		return &GetInboundUsersCountResponse{Count: 0}, err
	}

	return &GetInboundUsersCountResponse{Count: count}, nil
}

// DropUsersConnections RSTs all TCP connections for the given user IDs.
func (s *Service) DropUsersConnections(req *DropUsersConnectionsRequest) (*GenericResponse, error) {
	var allIPs []string
	for _, userID := range req.UserIDs {
		ips, err := s.xrayClient.GetStatsOnlineIpList("user>>>"+userID+">>>online", false)
		if err != nil {
			log.Warn().Err(err).Str("userId", userID).Msg("Failed to get user IPs for drop")
			continue
		}
		for _, ip := range ips {
			allIPs = append(allIPs, ip.IP)
		}
	}
	if len(allIPs) > 0 {
		if err := connkill.DropByIPs(allIPs); err != nil {
			log.Warn().Err(err).Strs("ips", allIPs).Msg("Failed to drop connections")
		}
	}
	return &GenericResponse{Success: true}, nil
}

// DropIps RSTs all TCP connections from the given IP addresses via SOCK_DESTROY.
func (s *Service) DropIps(req *DropIpsRequest) (*GenericResponse, error) {
	if err := connkill.DropByIPs(req.IPs); err != nil {
		log.Warn().Err(err).Msg("Failed to drop connections by IP")
	}
	return &GenericResponse{Success: true}, nil
}
