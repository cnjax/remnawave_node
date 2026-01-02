package xray

import "github.com/rs/zerolog/log"

// GenerateAPIConfig adds required API and policy configurations to the Xray config
func GenerateAPIConfig(config map[string]interface{}) map[string]interface{} {
	log.Info().Msg("GenerateAPIConfig called")

	if config == nil {
		log.Error().Msg("GenerateAPIConfig received nil config")
		config = make(map[string]interface{})
	}

	// Add stats configuration if not present
	if _, exists := config["stats"]; !exists {
		log.Info().Msg("Adding stats configuration")
		config["stats"] = map[string]interface{}{}
	}

	// Add API configuration
	config["api"] = map[string]interface{}{
		"tag": "api",
		"services": []string{
			"HandlerService",
			"StatsService",
			"RoutingService",
		},
	}

	// Add policy configuration for system stats
	policy := map[string]interface{}{
		"levels": map[string]interface{}{
			"0": map[string]interface{}{
				"statsUserUplink":   true,
				"statsUserDownlink": true,
			},
		},
		"system": map[string]interface{}{
			"statsInboundUplink":    true,
			"statsInboundDownlink":  true,
			"statsOutboundUplink":   true,
			"statsOutboundDownlink": true,
		},
	}
	config["policy"] = policy

	// Add API inbound if not present
	inbounds, ok := config["inbounds"].([]interface{})
	if !ok {
		inbounds = []interface{}{}
	}

	// Check if API inbound already exists
	hasAPIInbound := false
	for _, inbound := range inbounds {
		if inboundMap, ok := inbound.(map[string]interface{}); ok {
			if tag, ok := inboundMap["tag"].(string); ok && tag == "api" {
				hasAPIInbound = true
				break
			}
		}
	}

	if !hasAPIInbound {
		apiInbound := map[string]interface{}{
			"tag":      "api",
			"listen":   "127.0.0.1",
			"port":     61000,
			"protocol": "dokodemo-door",
			"settings": map[string]interface{}{
				"address": "127.0.0.1",
			},
		}
		inbounds = append([]interface{}{apiInbound}, inbounds...)
		config["inbounds"] = inbounds
	}

	// Add routing rules for API
	routing, ok := config["routing"].(map[string]interface{})
	if !ok {
		routing = map[string]interface{}{}
	}

	rules, ok := routing["rules"].([]interface{})
	if !ok {
		rules = []interface{}{}
	}

	// Check if API rule already exists
	hasAPIRule := false
	for _, rule := range rules {
		if ruleMap, ok := rule.(map[string]interface{}); ok {
			if tag, ok := ruleMap["outboundTag"].(string); ok && tag == "api" {
				hasAPIRule = true
				break
			}
		}
	}

	if !hasAPIRule {
		apiRule := map[string]interface{}{
			"inboundTag":  []string{"api"},
			"outboundTag": "api",
			"type":        "field",
		}
		rules = append([]interface{}{apiRule}, rules...)
	}

	routing["rules"] = rules
	config["routing"] = routing

	log.Info().Msg("GenerateAPIConfig completed successfully")
	return config
}
