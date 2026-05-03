package xray_client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	hysteriaAccount "github.com/xtls/xray-core/proxy/hysteria/account"
	"github.com/xtls/xray-core/proxy/shadowsocks"
	shadowsocks2022 "github.com/xtls/xray-core/proxy/shadowsocks_2022"
	"github.com/xtls/xray-core/proxy/trojan"
	"github.com/xtls/xray-core/proxy/vless"
)

// CipherType represents shadowsocks cipher types
type CipherType int32

const (
	CipherTypeUnknown           CipherType = 0
	CipherTypeAES128GCM         CipherType = 5
	CipherTypeAES256GCM         CipherType = 6
	CipherTypeCHACHA20POLY1305  CipherType = 7
	CipherTypeXCHACHA20POLY1305 CipherType = 8
	CipherTypeNone              CipherType = 9
)

// AddVlessUser adds a VLESS user to an inbound
func (c *Client) AddVlessUser(tag, username, uuid, flow string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	account := &vless.Account{
		Id:   uuid,
		Flow: flow,
	}

	user := &protocol.User{
		Level:   0,
		Email:   username,
		Account: serial.ToTypedMessage(account),
	}

	_, err := c.handler.AlterInbound(ctx, &command.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&command.AddUserOperation{
			User: user,
		}),
	})

	if err != nil {
		return fmt.Errorf("failed to add VLESS user: %w", err)
	}

	return nil
}

// AddTrojanUser adds a Trojan user to an inbound
func (c *Client) AddTrojanUser(tag, username, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	account := &trojan.Account{
		Password: password,
	}

	user := &protocol.User{
		Level:   0,
		Email:   username,
		Account: serial.ToTypedMessage(account),
	}

	_, err := c.handler.AlterInbound(ctx, &command.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&command.AddUserOperation{
			User: user,
		}),
	})

	if err != nil {
		return fmt.Errorf("failed to add Trojan user: %w", err)
	}

	return nil
}

// AddShadowsocksUser adds a Shadowsocks user to an inbound
func (c *Client) AddShadowsocksUser(tag, username, password string, cipherType CipherType, ivCheck bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	account := &shadowsocks.Account{
		Password:   password,
		CipherType: shadowsocks.CipherType(cipherType),
		IvCheck:    ivCheck,
	}

	user := &protocol.User{
		Level:   0,
		Email:   username,
		Account: serial.ToTypedMessage(account),
	}

	_, err := c.handler.AlterInbound(ctx, &command.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&command.AddUserOperation{
			User: user,
		}),
	})

	if err != nil {
		return fmt.Errorf("failed to add Shadowsocks user: %w", err)
	}

	return nil
}

// RemoveUser removes a user from an inbound.
// "not found" / code=5 errors are silently ignored; all other errors are returned.
func (c *Client) RemoveUser(tag, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.handler.AlterInbound(ctx, &command.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&command.RemoveUserOperation{
			Email: username,
		}),
	})

	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "not found") || strings.Contains(msg, "code = NotFound") {
			return nil
		}
		return fmt.Errorf("remove user %q from %q: %w", username, tag, err)
	}

	return nil
}

// InboundUser represents a user in an inbound
type InboundUser struct {
	Username string
	Email    string
	Level    uint32
}

// GetInboundUsers gets all users in an inbound via HandlerService.GetInboundUsers.
func (c *Client) GetInboundUsers(tag string) ([]InboundUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.handler.GetInboundUsers(ctx, &command.GetInboundUserRequest{Tag: tag})
	if err != nil {
		return nil, fmt.Errorf("GetInboundUsers %q: %w", tag, err)
	}

	users := make([]InboundUser, 0, len(resp.Users))
	for _, u := range resp.Users {
		users = append(users, InboundUser{
			Username: u.Email,
			Email:    u.Email,
			Level:    u.Level,
		})
	}
	return users, nil
}

// GetInboundUsersCount gets the count of users in an inbound
func (c *Client) GetInboundUsersCount(tag string) (int, error) {
	users, err := c.GetInboundUsers(tag)
	if err != nil {
		return 0, err
	}
	return len(users), nil
}

// AddShadowsocks2022User adds a Shadowsocks 2022 user to an inbound
func (c *Client) AddShadowsocks2022User(tag, username, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	account := &shadowsocks2022.Account{
		Key: key,
	}

	user := &protocol.User{
		Level:   0,
		Email:   username,
		Account: serial.ToTypedMessage(account),
	}

	_, err := c.handler.AlterInbound(ctx, &command.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&command.AddUserOperation{
			User: user,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to add Shadowsocks2022 user: %w", err)
	}
	return nil
}

// AddHysteriaUser adds a Hysteria user to an inbound
func (c *Client) AddHysteriaUser(tag, username, auth string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	account := &hysteriaAccount.Account{
		Auth: auth,
	}

	user := &protocol.User{
		Level:   0,
		Email:   username,
		Account: serial.ToTypedMessage(account),
	}

	_, err := c.handler.AlterInbound(ctx, &command.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&command.AddUserOperation{
			User: user,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to add Hysteria user: %w", err)
	}
	return nil
}

// RemoveOutbound removes an outbound from the Xray configuration
func (c *Client) RemoveOutbound(tag string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.handler.RemoveOutbound(ctx, &command.RemoveOutboundRequest{Tag: tag})
	if err != nil {
		return fmt.Errorf("failed to remove outbound: %w", err)
	}
	return nil
}
