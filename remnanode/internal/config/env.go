package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Parse SECRET_KEY first
	secretKey := os.Getenv("SECRET_KEY")
	payload, err := ParseNodePayload(secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SECRET_KEY: %w", err)
	}

	// Parse NODE_PORT (required)
	nodePortStr := os.Getenv("NODE_PORT")
	if nodePortStr == "" {
		return nil, fmt.Errorf("NODE_PORT environment variable is required")
	}
	nodePort, err := strconv.Atoi(nodePortStr)
	if err != nil {
		return nil, fmt.Errorf("NODE_PORT must be a valid integer: %w", err)
	}

	// XTLS_API_PORT is the canonical variable; fall back to XTLS_PORT for compatibility.
	xtlsIP := "127.0.0.1"
	xtlsPort := getEnvOrDefault("XTLS_API_PORT", getEnvOrDefault("XTLS_PORT", XrayGRPCPort))

	// Parse DISABLE_HASHED_SET_CHECK
	disableHashedSetCheck := false
	if val := os.Getenv("DISABLE_HASHED_SET_CHECK"); val != "" {
		disableHashedSetCheck = strings.ToLower(val) == "true" || val == "1"
	}

	// Get Xray core version
	xrayCoreVersion := os.Getenv("XRAY_CORE_VERSION")

	// Get Xray binary path (default: /usr/local/bin/rw-core)
	xrayBinaryPath := getEnvOrDefault("XRAY_BINARY_PATH", "/usr/local/bin/rw-core")

	return &Config{
		NodePort:              nodePort,
		XtlsIP:                xtlsIP,
		XtlsPort:              xtlsPort,
		DisableHashedSetCheck: disableHashedSetCheck,
		XrayCoreVersion:       xrayCoreVersion,
		XrayBinaryPath:        xrayBinaryPath,
		TLS: TLSConfig{
			CACertPem:   payload.CACertPem,
			NodeCertPem: payload.NodeCertPem,
			NodeKeyPem:  payload.NodeKeyPem,
		},
		JWTPublicKey: payload.JWTPublicKey,
	}, nil
}

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsDevelopment checks if running in development mode
func IsDevelopment() bool {
	env := os.Getenv("NODE_ENV")
	return env == "development" || env == "dev"
}
