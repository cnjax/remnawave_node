package state

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/pkg/hashedset"
)

// InboundHash represents a hash for an inbound configuration
type InboundHash struct {
	Tag        string `json:"tag"`
	Hash       string `json:"hash"`
	UsersCount int    `json:"usersCount"`
}

// HashesPayload represents the hashes data from the start request
type HashesPayload struct {
	EmptyConfig string        `json:"emptyConfig"`
	Inbounds    []InboundHash `json:"inbounds"`
}

// Manager manages the internal state of the node
type Manager struct {
	mu sync.RWMutex

	xrayConfig         map[string]interface{}
	emptyConfigHash    string
	inboundsHashMap    map[string]*hashedset.HashedSet
	xtlsConfigInbounds map[string]struct{}
}

// NewManager creates a new state manager
func NewManager() *Manager {
	return &Manager{
		inboundsHashMap:    make(map[string]*hashedset.HashedSet),
		xtlsConfigInbounds: make(map[string]struct{}),
	}
}

// GetXrayConfig returns the current Xray configuration
func (m *Manager) GetXrayConfig() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.xrayConfig == nil {
		log.Warn().Msg("GetXrayConfig: xrayConfig is nil, returning empty map")
		return make(map[string]interface{})
	}

	// Log config keys
	configKeys := make([]string, 0, len(m.xrayConfig))
	for k := range m.xrayConfig {
		configKeys = append(configKeys, k)
	}
	log.Info().Strs("configKeys", configKeys).Msg("GetXrayConfig returning config")

	return m.xrayConfig
}

// SetXrayConfig sets the current Xray configuration
func (m *Manager) SetXrayConfig(config map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.xrayConfig = config
}

// ExtractUsersFromConfig extracts users from the Xray configuration
func (m *Manager) ExtractUsersFromConfig(hashes HashesPayload, newConfig map[string]interface{}) {
	log.Info().Msg("ExtractUsersFromConfig called")

	if newConfig == nil {
		log.Error().Msg("ExtractUsersFromConfig received nil config")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	start := time.Now()

	// Clear internal state (inline cleanup without separate lock)
	log.Info().Msg("Cleaning up internal service before extraction")
	m.inboundsHashMap = make(map[string]*hashedset.HashedSet)
	m.xtlsConfigInbounds = make(map[string]struct{})

	m.emptyConfigHash = hashes.EmptyConfig
	m.xrayConfig = newConfig

	log.Info().
		Str("emptyConfigHash", hashes.EmptyConfig).
		Int("inboundsCount", len(hashes.Inbounds)).
		Msg("Starting user extraction from inbounds")

	// Get inbound configurations
	inbounds, ok := newConfig["inbounds"].([]interface{})
	if !ok {
		log.Warn().Msg("No inbounds found in config or invalid format")
		return
	}

	// Create a set of valid tags from hashes
	validTags := make(map[string]bool)
	for _, h := range hashes.Inbounds {
		validTags[h.Tag] = true
	}

	// Process each inbound
	for _, inboundRaw := range inbounds {
		inbound, ok := inboundRaw.(map[string]interface{})
		if !ok {
			continue
		}

		tag, _ := inbound["tag"].(string)
		if tag == "" || !validTags[tag] {
			continue
		}

		usersSet := hashedset.New()

		// Get settings.clients
		settings, ok := inbound["settings"].(map[string]interface{})
		if !ok {
			continue
		}

		clients, ok := settings["clients"].([]interface{})
		if !ok {
			continue
		}

		for _, clientRaw := range clients {
			client, ok := clientRaw.(map[string]interface{})
			if !ok {
				continue
			}

			// Extract user ID (could be "id" for VLESS or other fields)
			if id, ok := client["id"].(string); ok && id != "" {
				usersSet.Add(id)
			}
		}

		m.inboundsHashMap[tag] = usersSet
		m.xtlsConfigInbounds[tag] = struct{}{}

		log.Info().
			Str("tag", tag).
			Int("users", usersSet.Size()).
			Msg("Processed inbound")
	}

	// Log config keys to verify it's stored
	configKeys := make([]string, 0, len(m.xrayConfig))
	for k := range m.xrayConfig {
		configKeys = append(configKeys, k)
	}

	log.Info().
		Dur("duration", time.Since(start)).
		Strs("configKeys", configKeys).
		Int("totalInboundsProcessed", len(m.inboundsHashMap)).
		Msg("User extraction completed")
}

// IsNeedRestartCore checks if the core needs to be restarted based on config changes
func (m *Manager) IsNeedRestartCore(incomingHashes HashesPayload) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	start := time.Now()
	defer func() {
		log.Info().
			Dur("duration", time.Since(start)).
			Msg("Configuration hash check completed")
	}()

	// If we don't have a stored hash, we need to restart
	if m.emptyConfigHash == "" {
		return true
	}

	// Check if base configuration changed
	if incomingHashes.EmptyConfig != m.emptyConfigHash {
		log.Warn().Msg("Detected changes in Xray Core base configuration")
		return true
	}

	// Check if number of inbounds changed
	if len(incomingHashes.Inbounds) != len(m.inboundsHashMap) {
		log.Warn().Msg("Number of Xray Core inbounds has changed")
		return true
	}

	// Check each inbound's hash
	for tag, usersSet := range m.inboundsHashMap {
		var incomingInbound *InboundHash
		for i := range incomingHashes.Inbounds {
			if incomingHashes.Inbounds[i].Tag == tag {
				incomingInbound = &incomingHashes.Inbounds[i]
				break
			}
		}

		if incomingInbound == nil {
			log.Warn().
				Str("tag", tag).
				Msg("Inbound no longer exists in Xray Core configuration")
			return true
		}

		currentHash := usersSet.Hash64String()
		if currentHash != incomingInbound.Hash {
			log.Warn().
				Str("tag", tag).
				Str("current", currentHash).
				Str("incoming", incomingInbound.Hash).
				Msg("User configuration changed for inbound")
			return true
		}
	}

	log.Info().Msg("Xray Core configuration is up-to-date - no restart required")
	return false
}

// AddUserToInbound adds a user to an inbound's hash set
func (m *Manager) AddUserToInbound(inboundTag, userUUID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	usersSet, exists := m.inboundsHashMap[inboundTag]
	if !exists {
		log.Warn().
			Str("tag", inboundTag).
			Msg("Inbound not found in inboundsHashMap, creating new one")
		usersSet = hashedset.New()
		m.inboundsHashMap[inboundTag] = usersSet
	}

	usersSet.Add(userUUID)
}

// RemoveUserFromInbound removes a user from an inbound's hash set
func (m *Manager) RemoveUserFromInbound(inboundTag, userUUID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	usersSet, exists := m.inboundsHashMap[inboundTag]
	if !exists {
		return
	}

	usersSet.Delete(userUUID)

	// If the inbound has no more users, clean it up
	if usersSet.Size() == 0 {
		delete(m.xtlsConfigInbounds, inboundTag)
		delete(m.inboundsHashMap, inboundTag)
		log.Warn().
			Str("tag", inboundTag).
			Msg("Inbound has no users, clearing from inboundsHashMap")
	}
}

// GetXtlsConfigInbounds returns the set of inbound tags
func (m *Manager) GetXtlsConfigInbounds() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tags := make([]string, 0, len(m.xtlsConfigInbounds))
	for tag := range m.xtlsConfigInbounds {
		tags = append(tags, tag)
	}
	return tags
}

// AddXtlsConfigInbound adds an inbound tag
func (m *Manager) AddXtlsConfigInbound(inboundTag string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.xtlsConfigInbounds[inboundTag] = struct{}{}
}

// Cleanup clears all internal state
func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Info().Msg("Cleaning up internal service")

	m.inboundsHashMap = make(map[string]*hashedset.HashedSet)
	m.xtlsConfigInbounds = make(map[string]struct{})
	m.xrayConfig = nil
	m.emptyConfigHash = ""
}

// GetInboundUsersCount returns the number of users in an inbound
func (m *Manager) GetInboundUsersCount(tag string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if usersSet, exists := m.inboundsHashMap[tag]; exists {
		return usersSet.Size()
	}
	return 0
}
