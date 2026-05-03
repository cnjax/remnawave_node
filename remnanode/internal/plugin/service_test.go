package plugin

import (
	"errors"
	"testing"
)

func TestSyncEmptyPluginWithoutActivePluginIsNotAccepted(t *testing.T) {
	service := NewService(func() error {
		t.Fatal("stop xray should not be called")
		return nil
	})

	resp, err := service.Sync(&SyncRequest{Plugin: nil})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if resp.Accepted {
		t.Fatal("expected empty plugin sync without active plugin to be rejected")
	}
}

func TestSyncEmptyPluginWithActivePluginStopsXray(t *testing.T) {
	stopCalls := 0
	service := NewService(func() error {
		stopCalls++
		return nil
	})

	resp, err := service.Sync(&SyncRequest{Plugin: &PluginPayload{
		Config: map[string]interface{}{"enabled": true},
		UUID:   "f069a1f1-8f63-4a60-a454-b06ec8b73c40",
		Name:   "test-plugin",
	}})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if !resp.Accepted {
		t.Fatal("expected plugin sync to be accepted")
	}

	resp, err = service.Sync(&SyncRequest{Plugin: nil})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if !resp.Accepted {
		t.Fatal("expected cleanup sync to be accepted")
	}
	if stopCalls != 1 {
		t.Fatalf("expected stop xray to be called once, got %d", stopCalls)
	}
}

func TestSyncEmptyPluginReportsStopFailure(t *testing.T) {
	service := NewService(func() error {
		return errors.New("stop failed")
	})

	_, err := service.Sync(&SyncRequest{Plugin: &PluginPayload{
		Config: map[string]interface{}{"enabled": true},
		UUID:   "f069a1f1-8f63-4a60-a454-b06ec8b73c40",
		Name:   "test-plugin",
	}})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	resp, err := service.Sync(&SyncRequest{Plugin: nil})
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if resp.Accepted {
		t.Fatal("expected cleanup sync to be rejected when stop fails")
	}
}
