package xray_client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	handlerService "github.com/xtls/xray-core/app/proxyman/command"
	routerService "github.com/xtls/xray-core/app/router/command"
	statsService "github.com/xtls/xray-core/app/stats/command"

	"github.com/remnawave/remnanode/pkg/mtls"
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

// NewClient creates a new Xray gRPC client using ephemeral mTLS certs (pkg/mtls).
func NewClient(ip, port string) (*Client, error) {
	address := fmt.Sprintf("%s:%s", ip, port)

	creds, err := buildTLSCredentials()
	if err != nil {
		return nil, fmt.Errorf("build mTLS credentials: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(creds),
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

	log.Info().Str("address", address).Msg("Connected to Xray gRPC (mTLS)")
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.stats.GetSysStats(ctx, &statsService.SysStatsRequest{})
	return err == nil
}

// Reconnect attempts to reconnect to the Xray gRPC server using mTLS.
func (c *Client) Reconnect() error {
	if c.conn != nil {
		c.conn.Close()
	}

	creds, err := buildTLSCredentials()
	if err != nil {
		return fmt.Errorf("build mTLS credentials for reconnect: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, c.address,
		grpc.WithTransportCredentials(creds),
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

	log.Info().Str("address", c.address).Msg("Reconnected to Xray gRPC (mTLS)")
	return nil
}

// GetHandler returns the handler service client
func (c *Client) GetHandler() handlerService.HandlerServiceClient { return c.handler }

// GetStats returns the stats service client
func (c *Client) GetStats() statsService.StatsServiceClient { return c.stats }

// GetRouter returns the router service client
func (c *Client) GetRouter() routerService.RoutingServiceClient { return c.router }

// buildTLSCredentials constructs gRPC transport credentials from the ephemeral mTLS certs.
func buildTLSCredentials() (credentials.TransportCredentials, error) {
	certs := mtls.Get()

	clientCert, err := tls.X509KeyPair([]byte(certs.ClientCertPEM), []byte(certs.ClientKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("parse ephemeral client cert/key: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM([]byte(certs.CACertPEM)) {
		return nil, fmt.Errorf("parse ephemeral CA cert")
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caCertPool,
		ServerName:   mtls.ServerName,
		MinVersion:   tls.VersionTLS12,
	}
	return credentials.NewTLS(tlsCfg), nil
}
