package xray

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/internal/config"
	"github.com/remnawave/remnanode/internal/process"
	"github.com/remnawave/remnanode/internal/state"
	"github.com/remnawave/remnanode/internal/xray_client"
	"github.com/remnawave/remnanode/pkg/capabilities"
	"github.com/remnawave/remnanode/pkg/connkill"
	"github.com/remnawave/remnanode/pkg/sysinfo"
)

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

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
	systemInfo            *NodeSystemInfo
	disableHashedSetCheck bool
	hasCapNetAdmin        bool
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
		hasCapNetAdmin:        capabilities.HasCapNetAdmin(),
	}

	s.loadSystemInfo()
	return s
}

// loadSystemInfo reads static system information once at startup.
func (s *Service) loadSystemInfo() {
	info, err := sysinfo.GetSystemInfo()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get system info")
		return
	}

	s.systemInfo = &NodeSystemInfo{
		Arch:              info.Arch,
		CPUs:              info.CPUs,
		CPUModel:          info.CPUModel,
		MemoryTotal:       info.MemoryTotal,
		Hostname:          info.Hostname,
		Platform:          info.Platform,
		Release:           info.Release,
		Type:              info.Type,
		Version:           info.Version,
		NetworkInterfaces: info.NetworkInterfaces,
	}
}

// buildNodeSystem assembles the NodeSystem response object with live stats.
func (s *Service) buildNodeSystem() *NodeSystem {
	stats, err := sysinfo.GetSystemStats()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get system stats for response")
	}

	ns := &NodeSystem{
		Info:  s.systemInfo,
		Stats: stats,
	}
	if stats != nil {
		ns.Interface = stats.Interface
	}
	return ns
}

// StartXray starts the Xray process with the given configuration
func (s *Service) StartXray(req *StartXrayRequest, clientIP string) (resp *StartXrayResponse, err error) {
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
		_, err := s.xrayClient.GetSysStats()
		if err == nil {
			if !s.stateManager.IsNeedRestartCore(req.Internals.Hashes) {
				log.Info().Msg("Xray is healthy and config unchanged, skipping restart")
				return &StartXrayResponse{
					IsStarted:       true,
					Version:         &s.xrayVersion,
					System:          s.buildNodeSystem(),
					NodeInformation: &NodeInformation{Version: s.nodeVersion},
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

	log.Info().Msg("Generating full config with API settings")
	fullConfig := GenerateAPIConfig(
		req.XrayConfig,
		s.config.XtlsPort,
		s.hasCapNetAdmin,
	)

	log.Info().Msg("Extracting users from config")
	s.stateManager.ExtractUsersFromConfig(req.Internals.Hashes, fullConfig)

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
	isStarted := s.checkXrayHealth()

	if !isStarted {
		s.isXrayOnline = false
		errMsg := "Xray failed to start after retries"
		if diagnostics := s.processManager.Diagnostics(); diagnostics != "" {
			errMsg = errMsg + ": " + diagnostics
		}
		log.Error().
			Str("version", s.xrayVersion).
			Str("masterIP", clientIP).
			Str("diagnostics", s.processManager.Diagnostics()).
			Msg("Xray failed to start")

		return &StartXrayResponse{
			IsStarted:       false,
			Version:         &s.xrayVersion,
			Error:           &errMsg,
			System:          s.buildNodeSystem(),
			NodeInformation: &NodeInformation{Version: s.nodeVersion},
		}, nil
	}

	s.isXrayOnline = true
	log.Info().
		Str("version", s.xrayVersion).
		Str("masterIP", clientIP).
		Int("pid", s.processManager.GetPID()).
		Msg("Xray started successfully")

	return &StartXrayResponse{
		IsStarted:       true,
		Version:         &s.xrayVersion,
		System:          s.buildNodeSystem(),
		NodeInformation: &NodeInformation{Version: s.nodeVersion},
	}, nil
}

// StopXrayOptions controls optional behavior during shutdown.
type StopXrayOptions struct {
	// WithOnlineCheck: if true and CAP_NET_ADMIN is available, drop all online
	// users' TCP connections before stopping xray (matching TS withOnlineCheck).
	WithOnlineCheck bool
	// WithPluginCleanup: stub — will trigger plugin teardown once plugin module
	// is implemented (matching TS withPluginCleanup).
	WithPluginCleanup bool
}

// StopXray stops the Xray process.
// Call as StopXray() for a plain stop or StopXray(StopXrayOptions{...}) with options.
func (s *Service) StopXray(opts ...StopXrayOptions) (*StopXrayResponse, error) {
	var o StopXrayOptions
	if len(opts) > 0 {
		o = opts[0]
	}

	// Drop all online connections before stopping xray if requested.
	if o.WithOnlineCheck && s.hasCapNetAdmin && s.xrayClient != nil {
		s.dropAllOnlineConnections()
	}

	if o.WithPluginCleanup {
		// Plugin cleanup stub — will be wired to the plugin module in the future.
		log.Info().Msg("StopXray: withPluginCleanup requested (stub, no-op)")
	}

	if err := s.processManager.Stop(); err != nil {
		log.Error().Err(err).Msg("Failed to stop Xray process")
		return &StopXrayResponse{IsStopped: false}, nil
	}

	s.isXrayOnline = false
	s.stateManager.Cleanup()

	return &StopXrayResponse{IsStopped: true}, nil
}

// dropAllOnlineConnections retrieves all online users from xray and RSTs their TCP connections.
func (s *Service) dropAllOnlineConnections() {
	onlineUsers, err := s.xrayClient.GetAllOnlineUsers()
	if err != nil {
		log.Warn().Err(err).Msg("dropAllOnlineConnections: failed to get online users")
		return
	}

	var allIPs []string
	seen := make(map[string]struct{})
	for _, raw := range onlineUsers {
		// raw format: "user>>>userId>>>online"
		parts := strings.Split(raw, ">>>")
		if len(parts) < 2 {
			continue
		}
		userID := parts[1]
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}

		ips, err := s.xrayClient.GetStatsOnlineIpList("user>>>"+userID+">>>online", false)
		if err != nil {
			continue
		}
		for _, ip := range ips {
			allIPs = append(allIPs, ip.IP)
		}
	}

	if len(allIPs) == 0 {
		return
	}

	log.Info().Int("ips", len(allIPs)).Msg("Dropping all online connections before xray stop")
	if err := connkill.DropByIPs(allIPs); err != nil {
		log.Warn().Err(err).Msg("Failed to drop all online connections")
	}
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
		IsAlive:                  true,
		XrayInternalStatusCached: s.isXrayOnline,
		XrayVersion:              s.xrayVersion,
		NodeVersion:              s.nodeVersion,
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
