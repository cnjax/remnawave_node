package xray

import (
	"testing"
)

const testPort = "61000"

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

	fullConfig := GenerateAPIConfig(original, testPort, false)

	// --- api block: tag present, no listen field ---
	api, ok := fullConfig["api"].(map[string]interface{})
	if !ok {
		t.Fatalf("api config missing: %#v", fullConfig["api"])
	}
	if api["tag"] != apiTag {
		t.Fatalf("unexpected api tag: %#v", api["tag"])
	}
	if _, hasListen := api["listen"]; hasListen {
		t.Fatalf("api block must not have a listen field (§1 fix)")
	}

	// --- inbounds: API inbound prepended, original inbound still present ---
	inbounds, ok := fullConfig["inbounds"].([]interface{})
	if !ok || len(inbounds) != 2 {
		t.Fatalf("expected 2 inbounds (api + original), got %d", len(inbounds))
	}
	firstInbound, ok := inbounds[0].(map[string]interface{})
	if !ok || firstInbound["tag"] != apiInboundTag {
		t.Fatalf("first inbound must be %s, got %#v", apiInboundTag, inbounds[0])
	}
	if firstInbound["protocol"] != "dokodemo-door" {
		t.Fatalf("API inbound protocol must be dokodemo-door, got %v", firstInbound["protocol"])
	}

	// --- routing: rule 0 must be api rule ---
	routing, ok := fullConfig["routing"].(map[string]interface{})
	if !ok {
		t.Fatalf("routing config missing")
	}
	rules, ok := routing["rules"].([]interface{})
	if !ok || len(rules) == 0 {
		t.Fatalf("routing.rules must have at least one entry")
	}
	rule0, ok := rules[0].(map[string]interface{})
	if !ok {
		t.Fatalf("routing.rules[0] is not a map")
	}
	if rule0["outboundTag"] != apiTag {
		t.Fatalf("routing rule 0 outboundTag must be %s, got %v", apiTag, rule0["outboundTag"])
	}

	// --- policy preserved + flags set ---
	policy := fullConfig["policy"].(map[string]interface{})
	levels := policy["levels"].(map[string]interface{})
	level0 := levels["0"].(map[string]interface{})
	if level0["handshake"] != float64(4) {
		t.Fatalf("existing level 0 policy was not preserved: %#v", level0)
	}
	if level0["statsUserUplink"] != true || level0["statsUserDownlink"] != true {
		t.Fatalf("stats uplink/downlink flags not set: %#v", level0)
	}
	// hasCapNetAdmin=false → statsUserOnline must be false
	if level0["statsUserOnline"] != false {
		t.Fatalf("statsUserOnline should be false when hasCapNetAdmin=false: %#v", level0)
	}
}

func TestGenerateAPIConfigCapNetAdmin(t *testing.T) {
	cfg := GenerateAPIConfig(nil, testPort, true)
	policy := cfg["policy"].(map[string]interface{})
	level0 := policy["levels"].(map[string]interface{})["0"].(map[string]interface{})
	if level0["statsUserOnline"] != true {
		t.Fatalf("statsUserOnline should be true when hasCapNetAdmin=true: %#v", level0)
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

	_ = GenerateAPIConfig(original, testPort, false)

	if _, exists := original["api"]; exists {
		t.Fatalf("GenerateAPIConfig mutated input api")
	}
	if _, exists := original["stats"]; exists {
		t.Fatalf("GenerateAPIConfig mutated input stats")
	}
	if _, exists := original["routing"]; exists {
		t.Fatalf("GenerateAPIConfig mutated input routing")
	}
	if _, exists := original["inbounds"]; exists {
		t.Fatalf("GenerateAPIConfig mutated input inbounds (nil inbounds should not be set)")
	}
	level0 := original["policy"].(map[string]interface{})["levels"].(map[string]interface{})["0"].(map[string]interface{})
	if _, exists := level0["statsUserUplink"]; exists {
		t.Fatalf("GenerateAPIConfig mutated nested input policy")
	}
}
