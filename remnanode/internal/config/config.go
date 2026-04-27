package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

// Config holds all application configuration
type Config struct {
	NodePort              int
	XtlsIP                string
	XtlsPort              string
	DisableHashedSetCheck bool
	XrayCoreVersion       string
	XrayBinaryPath        string
	TLS                   TLSConfig
	JWTPublicKey          string
}

// TLSConfig holds TLS certificate configuration
type TLSConfig struct {
	CACertPem   string
	NodeCertPem string
	NodeKeyPem  string
}

// Constants for internal ports
const (
	XrayInternalAPIPort = 61001
	SupervisordPort     = 61002
	XrayGRPCPort        = "61000"
)

// GetTLSConfig creates a tls.Config for the main HTTPS server with mTLS
func (c *Config) GetTLSConfig() (*tls.Config, error) {
	// Load server certificate and key
	cert, err := tls.X509KeyPair([]byte(c.TLS.NodeCertPem), []byte(c.TLS.NodeKeyPem))
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// Create CA certificate pool
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM([]byte(c.TLS.CACertPem)) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
