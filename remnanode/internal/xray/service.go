package xray

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/internal/config"
	"github.com/remnawave/remnanode/internal/process"
	"github.com/remnawave/remnanode/internal/state"
	"github.com/remnawave/remnanode/internal/xray_client"
	"github.com/remnawave/remnanode/pkg/sysinfo"
)

const (
	maxRetries    = 10
	retryInterval = 2 * time.Second
)

// Service handles Xray process lifecycle
type Service struct {
	mu sync.Mutex

	xrayClient     *xray_client.Client
	processManager *process.Manager
	stateManager   *state.Manager
	config         *config.Config

	xrayVersion           string
	nodeVersion           string
	isXrayOnline          bool
	isProcessing          bool
	systemStats           *SystemInfo
	disableHashedSetCheck bool
}

// NewService creates a new Xray service
func NewService(
	xrayClient *xray_client.Client,
	processManager *process.Manager,
	stateManager *state.Manager,
	cfg *config.Config,
	nodeVersion string,
) *Service {
	s := &Service{
		xrayClient:            xrayClient,
		processManager:        processManager,
		stateManager:          stateManager,
		config:                cfg,
		nodeVersion:           nodeVersion,
		xrayVersion:           cfg.XrayCoreVersion,
		disableHashedSetCheck: cfg.DisableHashedSetCheck,
	}

	// Get system stats on initialization
	s.loadSystemStats()

	return s
}

// loadSystemStats loads system information
func (s *Service) loadSystemStats() {
	stats, err := sysinfo.GetSystemStats()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get system stats")
		return
	}

	s.systemStats = &SystemInfo{
		CPUCores:    stats.CPUCores,
		CPUModel:    stats.CPUModel,
		MemoryTotal: stats.MemoryTotal,
	}
}

// StartXray starts the Xray process with the given configuration
func (s *Service) StartXray(req *StartXrayRequest, clientIP string) (resp *StartXrayResponse, err error) {
	// Recover from any panic
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("PANIC in StartXray")
			errMsg := "Internal panic error"
			resp = &StartXrayResponse{
				IsStarted:       false,
				Error:           &errMsg,
				NodeInformation: &NodeInformation{Version: s.nodeVersion},
			}
			err = nil
		}
	}()

	log.Info().Str("clientIP", clientIP).Msg("StartXray called")

	s.mu.Lock()
	if s.isProcessing {
		s.mu.Unlock()
		log.Warn().Msg("Request already in progress")
		errMsg := "Request already in progress"
		return &StartXrayResponse{
			IsStarted:       false,
			Version:         &s.xrayVersion,
			Error:           &errMsg,
			NodeInformation: &NodeInformation{Version: s.nodeVersion},
		}, nil
	}
	s.isProcessing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isProcessing = false
		s.mu.Unlock()
	}()

	start := time.Now()
	defer func() {
		log.Info().
			Dur("duration", time.Since(start)).
			Msg("Attempt to start XTLS completed")
	}()

	log.Info().Msg("StartXray: passed processing check")

	// Check if restart is needed
	if s.isXrayOnline && !s.disableHashedSetCheck && !req.Internals.ForceRestart && s.xrayClient != nil {
		// Check if Xray is healthy
		_, err := s.xrayClient.GetSysStats()
		if err == nil {
			// Xray is healthy, check if config changed
			if !s.stateManager.IsNeedRestartCore(req.Internals.Hashes) {
				log.Info().Msg("Xray is healthy and config unchanged, skipping restart")
				return &StartXrayResponse{
					IsStarted:         true,
					Version:           &s.xrayVersion,
					SystemInformation: s.systemStats,
					NodeInformation:   &NodeInformation{Version: s.nodeVersion},
				}, nil
			}
		} else {
			s.isXrayOnline = false
			log.Warn().Err(err).Msg("Xray Core health check failed, restarting...")
		}
	}

	if req.Internals.ForceRestart {
		log.Warn().Msg("Force restart requested")
	}

	// Generate full config with API settings
	log.Info().Msg("Generating full config with API settings")
	fullConfig := GenerateAPIConfig(req.XrayConfig)

	// Extract users from config and store it (for /internal/get-config endpoint)
	log.Info().Msg("Extracting users from config")
	s.stateManager.ExtractUsersFromConfig(req.Internals.Hashes, fullConfig)

	// Restart Xray process directly
	log.Info().Msg("Restarting Xray process")
	if err := s.processManager.Restart(); err != nil {
		log.Error().Err(err).Msg("Failed to restart Xray process")
		errMsg := err.Error()
		return &StartXrayResponse{
			IsStarted:       false,
			Error:           &errMsg,
			NodeInformation: &NodeInformation{Version: s.nodeVersion},
		}, nil
	}

	log.Info().Msg("Xray process started, checking health via gRPC")
	// Check if Xray started successfully
	isStarted := s.checkXrayHealth()

	if !isStarted {
		s.isXrayOnline = false
		log.Error().
			Str("version", s.xrayVersion).
			Str("masterIP", clientIP).
			Msg("Xray failed to start")

		errMsg := "Xray failed to start after retries"
		return &StartXrayResponse{
			IsStarted:         false,
			Version:           &s.xrayVersion,
			Error:             &errMsg,
			SystemInformation: s.systemStats,
			NodeInformation:   &NodeInformation{Version: s.nodeVersion},
		}, nil
	}

	s.isXrayOnline = true
	log.Info().
		Str("version", s.xrayVersion).
		Str("masterIP", clientIP).
		Int("pid", s.processManager.GetPID()).
		Msg("Xray started successfully")

	return &StartXrayResponse{
		IsStarted:         true,
		Version:           &s.xrayVersion,
		SystemInformation: s.systemStats,
		NodeInformation:   &NodeInformation{Version: s.nodeVersion},
	}, nil
}

// StopXray stops the Xray process
func (s *Service) StopXray() (*StopXrayResponse, error) {
	if err := s.processManager.Stop(); err != nil {
		log.Error().Err(err).Msg("Failed to stop Xray process")
		return &StopXrayResponse{IsStopped: false}, nil
	}

	s.isXrayOnline = false
	s.stateManager.Cleanup()

	return &StopXrayResponse{IsStopped: true}, nil
}

// GetStatus returns the current Xray status
func (s *Service) GetStatus() (*GetXrayStatusResponse, error) {
	isRunning := s.processManager.IsRunning() && s.checkXrayHealth()

	return &GetXrayStatusResponse{
		IsRunning: isRunning,
		Version:   s.xrayVersion,
	}, nil
}

// GetNodeHealthCheck returns the node health status
func (s *Service) GetNodeHealthCheck() (*GetNodeHealthCheckResponse, error) {
	return &GetNodeHealthCheckResponse{
		IsHealthy:    true,
		IsXrayOnline: s.isXrayOnline,
		XrayVersion:  s.xrayVersion,
		NodeVersion:  s.nodeVersion,
	}, nil
}

// checkXrayHealth checks if Xray is running and healthy via gRPC
func (s *Service) checkXrayHealth() bool {
	if s.xrayClient == nil {
		log.Error().Msg("Xray client is nil, cannot check health")
		return false
	}

	for i := 0; i < maxRetries; i++ {
		_, err := s.xrayClient.GetSysStats()
		if err == nil {
			log.Info().Int("attempt", i+1).Msg("Xray health check succeeded")
			return true
		}

		log.Debug().
			Err(err).
			Int("attempt", i+1).
			Int("remaining", maxRetries-i-1).
			Msg("Xray health check failed, retrying...")

		time.Sleep(retryInterval)
	}

	log.Error().Msg("Failed to get Xray internal status after all retries")
	return false
}

// Cleanup cleans up resources
func (s *Service) Cleanup() {
	s.processManager.Stop()
	s.stateManager.Cleanup()
}
