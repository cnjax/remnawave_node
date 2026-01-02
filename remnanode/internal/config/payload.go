package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// NodePayload represents the decoded SECRET_KEY JSON structure
type NodePayload struct {
	CACertPem    string `json:"caCertPem"`
	JWTPublicKey string `json:"jwtPublicKey"`
	NodeCertPem  string `json:"nodeCertPem"`
	NodeKeyPem   string `json:"nodeKeyPem"`
}

// ParseNodePayload decodes the base64-encoded SECRET_KEY and extracts the payload
func ParseNodePayload(secretKey string) (*NodePayload, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("SECRET_KEY is empty")
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SECRET_KEY from base64: %w", err)
	}

	// Parse JSON
	var payload NodePayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse SECRET_KEY JSON: %w", err)
	}

	// Validate required fields
	if payload.CACertPem == "" {
		return nil, fmt.Errorf("caCertPem is missing in SECRET_KEY")
	}
	if payload.JWTPublicKey == "" {
		return nil, fmt.Errorf("jwtPublicKey is missing in SECRET_KEY")
	}
	if payload.NodeCertPem == "" {
		return nil, fmt.Errorf("nodeCertPem is missing in SECRET_KEY")
	}
	if payload.NodeKeyPem == "" {
		return nil, fmt.Errorf("nodeKeyPem is missing in SECRET_KEY")
	}

	return &payload, nil
}
