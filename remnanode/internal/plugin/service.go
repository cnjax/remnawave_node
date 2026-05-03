package plugin

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type Service struct {
	mu         sync.Mutex
	activeUUID string
	activeName string
	stopXrayFn func() error
}

func NewService(stopXrayFn func() error) *Service {
	return &Service{stopXrayFn: stopXrayFn}
}

func (s *Service) Sync(req *SyncRequest) (*SyncResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.Plugin == nil {
		if s.activeUUID == "" {
			log.Info().Msg("[PLUGIN] Sync received empty plugin; no active plugin to clean up")
			return &SyncResponse{Accepted: false}, nil
		}

		log.Info().
			Str("plugin_uuid", s.activeUUID).
			Str("plugin_name", s.activeName).
			Msg("[PLUGIN] Sync received empty plugin; cleaning active plugin and stopping xray")

		s.activeUUID = ""
		s.activeName = ""
		if s.stopXrayFn != nil {
			if err := s.stopXrayFn(); err != nil {
				log.Error().Err(err).Msg("[PLUGIN] Failed to stop xray during plugin cleanup")
				return &SyncResponse{Accepted: false}, nil
			}
		}

		return &SyncResponse{Accepted: true}, nil
	}

	if req.Plugin.Config == nil || req.Plugin.UUID == "" || req.Plugin.Name == "" {
		log.Warn().
			Bool("has_config", req.Plugin.Config != nil).
			Bool("has_uuid", req.Plugin.UUID != "").
			Bool("has_name", req.Plugin.Name != "").
			Msg("[PLUGIN] Invalid plugin sync payload")
		return &SyncResponse{Accepted: false}, nil
	}

	s.activeUUID = req.Plugin.UUID
	s.activeName = req.Plugin.Name
	log.Info().
		Str("plugin_uuid", req.Plugin.UUID).
		Str("plugin_name", req.Plugin.Name).
		Int("config_keys", len(req.Plugin.Config)).
		Msg("[PLUGIN] Plugin sync accepted")

	return &SyncResponse{Accepted: true}, nil
}
