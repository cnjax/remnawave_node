package xray

import "github.com/rs/zerolog/log"

// GenerateAPIConfig adds required API and policy configurations to the Xray config
func GenerateAPIConfig(config map[string]interface{}) map[string]interface{} {
	log.Info().Msg("GenerateAPIConfig called")

	if config == nil {
		log.Error().Msg("GenerateAPIConfig received nil config")
		config = make(map[string]interface{})
	}

	fullConfig := cloneMap(config)

	fullConfig["stats"] = map[string]interface{}{}
	fullConfig["api"] = map[string]interface{}{
		"services": []string{
			"HandlerService",
			"StatsService",
			"RoutingService",
		},
		"listen": "127.0.0.1:61000",
		"tag":    "REMNAWAVE_API",
	}

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
	level0["statsUserOnline"] = false

	policy := map[string]interface{}{
		"levels": map[string]interface{}{
			"0": level0,
		},
		"system": map[string]interface{}{
			"statsInboundUplink":    true,
			"statsInboundDownlink":  true,
			"statsOutboundUplink":   true,
			"statsOutboundDownlink": true,
		},
	}
	fullConfig["policy"] = policy

	log.Info().Msg("GenerateAPIConfig completed successfully")
	return fullConfig
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
