package xray

import (
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/pkg/mtls"
)

const (
	apiTag        = "REMNAWAVE_API"
	apiInboundTag = "REMNAWAVE_API_INBOUND"
)

// GenerateAPIConfig injects the required API inbound (dokodemo-door + mTLS),
// routing rule, stats block, and policy into the supplied Xray config map.
// xtlsPort must be a numeric string (e.g. "61000").
// hasCapNetAdmin controls the policy.levels['0'].statsUserOnline flag.
// Ephemeral mTLS certs from pkg/mtls are used (not the node certs from SECRET_KEY).
func GenerateAPIConfig(
	config map[string]interface{},
	xtlsPort string,
	hasCapNetAdmin bool,
) map[string]interface{} {
	log.Info().Msg("GenerateAPIConfig called")

	if config == nil {
		log.Error().Msg("GenerateAPIConfig received nil config")
		config = make(map[string]interface{})
	}

	fullConfig := cloneMap(config)

	// --- stats block ---
	fullConfig["stats"] = map[string]interface{}{}

	// --- api block (no listen field; access is via the inbound below) ---
	fullConfig["api"] = map[string]interface{}{
		"services": []string{
			"HandlerService",
			"StatsService",
			"RoutingService",
		},
		"tag": apiTag,
	}

	// --- policy ---
	level0 := map[string]interface{}{}
	if policyConfig, ok := fullConfig["policy"].(map[string]interface{}); ok {
		if levels, ok := policyConfig["levels"].(map[string]interface{}); ok {
			if existingLevel0, ok := levels["0"].(map[string]interface{}); ok {
				level0 = cloneMap(existingLevel0)
			}
		}
	}

	level0["statsUserUplink"] = true
	level0["statsUserDownlink"] = true
	level0["statsUserOnline"] = hasCapNetAdmin

	fullConfig["policy"] = map[string]interface{}{
		"levels": map[string]interface{}{"0": level0},
		"system": map[string]interface{}{
			"statsInboundUplink":    true,
			"statsInboundDownlink":  true,
			"statsOutboundUplink":   true,
			"statsOutboundDownlink": true,
		},
	}

	// --- API inbound: dokodemo-door + mTLS (ephemeral certs from pkg/mtls) ---
	// Server presents the ephemeral server cert (has internal.remnawave.local SAN so
	// Go's strict x509 hostname verification in the gRPC client passes).
	// The CA cert is added with usage=verify so xray requires the client cert too.
	certs := mtls.Get()
	port, _ := parseInt(xtlsPort)
	apiInbound := map[string]interface{}{
		"tag":      apiInboundTag,
		"listen":   "127.0.0.1",
		"port":     port,
		"protocol": "dokodemo-door",
		"settings": map[string]interface{}{
			"address": "127.0.0.1",
		},
		"streamSettings": map[string]interface{}{
			"security": "tls",
			"tlsSettings": map[string]interface{}{
				"serverName":        mtls.ServerName,
				"alpn":              []string{"h2"},
				"disableSystemRoot": true,
				"rejectUnknownSni":  true,
				"certificates": []interface{}{
					map[string]interface{}{
						"certificate": pemToLines(certs.ServerCertPEM),
						"key":         pemToLines(certs.ServerKeyPEM),
					},
					map[string]interface{}{
						"usage":       "verify",
						"certificate": pemToLines(certs.CACertPEM),
					},
				},
			},
		},
	}

	// Prepend API inbound
	existingInbounds, _ := fullConfig["inbounds"].([]interface{})
	fullConfig["inbounds"] = append([]interface{}{apiInbound}, existingInbounds...)

	// --- routing rule: REMNAWAVE_API_INBOUND → REMNAWAVE_API as rule 0 ---
	apiRule := map[string]interface{}{
		"type":        "field",
		"inboundTag":  []string{apiInboundTag},
		"outboundTag": apiTag,
	}

	var existingRules []interface{}
	if routing, ok := fullConfig["routing"].(map[string]interface{}); ok {
		existingRules, _ = routing["rules"].([]interface{})
	}
	newRules := append([]interface{}{apiRule}, existingRules...)

	existingRouting, ok := fullConfig["routing"].(map[string]interface{})
	if ok {
		routingCopy := cloneMap(existingRouting)
		routingCopy["rules"] = newRules
		fullConfig["routing"] = routingCopy
	} else {
		fullConfig["routing"] = map[string]interface{}{
			"rules": newRules,
		}
	}

	log.Info().Msg("GenerateAPIConfig completed successfully")
	return fullConfig
}

// pemToLines splits a PEM string into individual lines (xray inline cert format).
func pemToLines(pem string) []string {
	lines := strings.Split(strings.TrimSpace(pem), "\n")
	result := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}

func parseInt(s string) (int, bool) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = cloneValue(v)
	}
	return dst
}

func cloneSlice(src []interface{}) []interface{} {
	dst := make([]interface{}, len(src))
	for i, v := range src {
		dst[i] = cloneValue(v)
	}
	return dst
}

func cloneValue(v interface{}) interface{} {
	switch typed := v.(type) {
	case map[string]interface{}:
		return cloneMap(typed)
	case []interface{}:
		return cloneSlice(typed)
	default:
		return typed
	}
}
