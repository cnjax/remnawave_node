package xray_client

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	handlerService "github.com/xtls/xray-core/app/proxyman/command"
	routerService "github.com/xtls/xray-core/app/router/command"
	statsService "github.com/xtls/xray-core/app/stats/command"
)

const (
	// MaxMessageSize is the maximum gRPC message size (100MB)
	MaxMessageSize = 100 * 1024 * 1024
)

// Client wraps the gRPC connection to Xray core
type Client struct {
	conn    *grpc.ClientConn
	address string

	handler handlerService.HandlerServiceClient
	stats   statsService.StatsServiceClient
	router  routerService.RoutingServiceClient
}

// NewClient creates a new Xray gRPC client
func NewClient(ip, port string) (*Client, error) {
	address := fmt.Sprintf("%s:%s", ip, port)

	// Create gRPC connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(MaxMessageSize),
			grpc.MaxCallSendMsgSize(MaxMessageSize),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Xray gRPC: %w", err)
	}

	client := &Client{
		conn:    conn,
		address: address,
		handler: handlerService.NewHandlerServiceClient(conn),
		stats:   statsService.NewStatsServiceClient(conn),
		router:  routerService.NewRoutingServiceClient(conn),
	}

	log.Info().Str("address", address).Msg("Connected to Xray gRPC")

	return client, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	if c.conn == nil {
		return false
	}
	// Try to get system stats as a health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.stats.GetSysStats(ctx, &statsService.SysStatsRequest{})
	return err == nil
}

// Reconnect attempts to reconnect to the Xray gRPC server
func (c *Client) Reconnect() error {
	if c.conn != nil {
		c.conn.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, c.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(MaxMessageSize),
			grpc.MaxCallSendMsgSize(MaxMessageSize),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to reconnect to Xray gRPC: %w", err)
	}

	c.conn = conn
	c.handler = handlerService.NewHandlerServiceClient(conn)
	c.stats = statsService.NewStatsServiceClient(conn)
	c.router = routerService.NewRoutingServiceClient(conn)

	log.Info().Str("address", c.address).Msg("Reconnected to Xray gRPC")

	return nil
}

// GetHandler returns the handler service client
func (c *Client) GetHandler() handlerService.HandlerServiceClient {
	return c.handler
}

// GetStats returns the stats service client
func (c *Client) GetStats() statsService.StatsServiceClient {
	return c.stats
}

// GetRouter returns the router service client
func (c *Client) GetRouter() routerService.RoutingServiceClient {
	return c.router
}
