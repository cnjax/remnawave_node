package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var semverCoercePattern = regexp.MustCompile(`\d+(?:\.\d+){0,2}`)

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
	xrayCoreVersion := coerceSemver(os.Getenv("XRAY_CORE_VERSION"))

	// Get Xray binary path (default: /usr/local/bin/rw-core)
	xrayBinaryPath := getEnvOrDefault("XRAY_BINARY_PATH", "/usr/local/bin/rw-core")

	// INTERNAL_REST_TOKEN is required — protects /internal/* endpoints
	internalRestToken := os.Getenv("INTERNAL_REST_TOKEN")
	if internalRestToken == "" {
		return nil, fmt.Errorf("INTERNAL_REST_TOKEN environment variable is required")
	}

	return &Config{
		NodePort:              nodePort,
		XtlsIP:                xtlsIP,
		XtlsPort:              xtlsPort,
		DisableHashedSetCheck: disableHashedSetCheck,
		XrayCoreVersion:       xrayCoreVersion,
		XrayBinaryPath:        xrayBinaryPath,
		InternalRestToken:     internalRestToken,
		TLS: TLSConfig{
			CACertPem:   payload.CACertPem,
			NodeCertPem: payload.NodeCertPem,
			NodeKeyPem:  payload.NodeKeyPem,
		},
		JWTPublicKey: payload.JWTPublicKey,
	}, nil
}

func coerceSemver(raw string) string {
	version := semverCoercePattern.FindString(raw)
	if version == "" {
		return ""
	}

	parts := strings.Split(version, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	return strings.Join(parts[:3], ".")
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
