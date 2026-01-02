package xray_client

import (
	"context"
	"fmt"
	"time"

	"github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/proxy/shadowsocks"
	"github.com/xtls/xray-core/proxy/trojan"
	"github.com/xtls/xray-core/proxy/vless"
)

// CipherType represents shadowsocks cipher types
type CipherType int32

const (
	CipherTypeUnknown         CipherType = 0
	CipherTypeAES128GCM       CipherType = 5
	CipherTypeAES256GCM       CipherType = 6
	CipherTypeCHACHA20POLY1305 CipherType = 7
	CipherTypeXCHACHA20POLY1305 CipherType = 8
	CipherTypeNone            CipherType = 9
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

// RemoveUser removes a user from an inbound
func (c *Client) RemoveUser(tag, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.handler.AlterInbound(ctx, &command.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&command.RemoveUserOperation{
			Email: username,
		}),
	})

	// Ignore "not found" errors as the user might not exist
	if err != nil {
		// TODO: Check if error is "user not found" and ignore it
		return nil
	}

	return nil
}

// InboundUser represents a user in an inbound
type InboundUser struct {
	Username string
	Email    string
	Level    uint32
}

// GetInboundUsers gets all users in an inbound
// Note: This requires Xray core to support GetInboundUsers which may not be available in all versions
func (c *Client) GetInboundUsers(tag string) ([]InboundUser, error) {
	// This is a simplified implementation
	// The actual implementation depends on Xray core version and available APIs
	return []InboundUser{}, nil
}

// GetInboundUsersCount gets the count of users in an inbound
func (c *Client) GetInboundUsersCount(tag string) (int, error) {
	users, err := c.GetInboundUsers(tag)
	if err != nil {
		return 0, err
	}
	return len(users), nil
}
