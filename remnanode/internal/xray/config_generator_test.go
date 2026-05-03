package xray

import "testing"

func TestGenerateAPIConfigMatchesTSRuntimeShape(t *testing.T) {
	original := map[string]interface{}{
		"inbounds": []interface{}{
			map[string]interface{}{
				"tag":      "public",
				"protocol": "vless",
			},
		},
		"policy": map[string]interface{}{
			"levels": map[string]interface{}{
				"0": map[string]interface{}{
					"handshake": float64(4),
				},
			},
		},
	}

	fullConfig := GenerateAPIConfig(original)

	api, ok := fullConfig["api"].(map[string]interface{})
	if !ok {
		t.Fatalf("api config missing or invalid: %#v", fullConfig["api"])
	}
	if api["tag"] != "REMNAWAVE_API" {
		t.Fatalf("unexpected api tag: %#v", api["tag"])
	}
	if api["listen"] != "127.0.0.1:61000" {
		t.Fatalf("unexpected api listen: %#v", api["listen"])
	}

	inbounds := fullConfig["inbounds"].([]interface{})
	if len(inbounds) != 1 {
		t.Fatalf("GenerateAPIConfig must not inject API inbound, got %d inbounds", len(inbounds))
	}
	if _, exists := fullConfig["routing"]; exists {
		t.Fatalf("GenerateAPIConfig must not inject API routing rules")
	}

	policy := fullConfig["policy"].(map[string]interface{})
	levels := policy["levels"].(map[string]interface{})
	level0 := levels["0"].(map[string]interface{})
	if level0["handshake"] != float64(4) {
		t.Fatalf("existing level 0 policy was not preserved: %#v", level0)
	}
	if level0["statsUserUplink"] != true || level0["statsUserDownlink"] != true || level0["statsUserOnline"] != false {
		t.Fatalf("stats policy flags are not TS-compatible: %#v", level0)
	}
}

func TestGenerateAPIConfigDoesNotMutateInput(t *testing.T) {
	original := map[string]interface{}{
		"policy": map[string]interface{}{
			"levels": map[string]interface{}{
				"0": map[string]interface{}{},
			},
		},
	}

	_ = GenerateAPIConfig(original)

	if _, exists := original["api"]; exists {
		t.Fatalf("GenerateAPIConfig mutated input api")
	}
	if _, exists := original["stats"]; exists {
		t.Fatalf("GenerateAPIConfig mutated input stats")
	}
	level0 := original["policy"].(map[string]interface{})["levels"].(map[string]interface{})["0"].(map[string]interface{})
	if _, exists := level0["statsUserUplink"]; exists {
		t.Fatalf("GenerateAPIConfig mutated nested input policy")
	}
}
