package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/internal/config"
	"github.com/remnawave/remnanode/internal/handler"
	internalapi "github.com/remnawave/remnanode/internal/internal_api"
	"github.com/remnawave/remnanode/internal/plugin"
	"github.com/remnawave/remnanode/internal/process"
	"github.com/remnawave/remnanode/internal/server"
	"github.com/remnawave/remnanode/internal/state"
	"github.com/remnawave/remnanode/internal/stats"
	"github.com/remnawave/remnanode/internal/vision"
	"github.com/remnawave/remnanode/internal/xray"
	"github.com/remnawave/remnanode/internal/xray_client"
)

const (
	nodeVersion = "2.7.0"
)

func main() {
	// Setup logging
	setupLogging()

	log.Info().Msg("Starting Remnawave Node...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().
		Int("port", cfg.NodePort).
		Str("xtls_ip", cfg.XtlsIP).
		Str("xtls_port", cfg.XtlsPort).
		Str("xray_binary", cfg.XrayBinaryPath).
		Msg("Configuration loaded")

	// Create Xray gRPC client
	xrayClient, err := xray_client.NewClient(cfg.XtlsIP, cfg.XtlsPort)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create Xray client (will retry on requests)")
	}

	// Create process manager for Xray (replaces supervisord)
	configURL := fmt.Sprintf("http://127.0.0.1:%d/internal/get-config", config.XrayInternalAPIPort)
	processManager := process.NewManager(cfg.XrayBinaryPath, configURL)

	// Check if Xray binary exists
	if err := processManager.CheckBinary(); err != nil {
		log.Warn().Err(err).Msg("Xray binary check failed (will fail on start requests)")
	}

	// Create state manager
	stateManager := state.NewManager()

	// Create services
	handlerService := handler.NewService(xrayClient, stateManager, cfg.DisableHashedSetCheck)
	statsService := stats.NewService(xrayClient)
	xrayService := xray.NewService(xrayClient, processManager, stateManager, cfg, nodeVersion)
	pluginService := plugin.NewService(func() error {
		_, err := xrayService.StopXray()
		return err
	})
	visionService := vision.NewService(xrayClient)
	internalAPIService := internalapi.NewService(stateManager)

	// Create handlers
	handlerHandler := handler.NewHandler(handlerService)
	statsHandler := stats.NewHandler(statsService)
	xrayHandler := xray.NewHandler(xrayService)
	pluginHandler := plugin.NewHandler(pluginService)
	visionHandler := vision.NewHandler(visionService)
	internalAPIHandler := internalapi.NewHandler(internalAPIService)

	// Create server with services
	services := &server.Services{
		Handler:     handlerHandler,
		Stats:       statsHandler,
		Xray:        xrayHandler,
		Plugin:      pluginHandler,
		Vision:      visionHandler,
		InternalAPI: internalAPIHandler,
	}

	srv, err := server.New(cfg, services)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create server")
	}

	// Start server
	if err := srv.Start(); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}

	log.Info().
		Int("main_port", cfg.NodePort).
		Int("internal_port", config.XrayInternalAPIPort).
		Str("version", nodeVersion).
		Msg("Remnawave Node started successfully")

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Drain in-flight HTTP requests first so gRPC calls can complete normally.
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
	}

	// Now safe to stop xray and clean up state/gRPC.
	processManager.Cleanup()
	stateManager.Cleanup()
	if xrayClient != nil {
		xrayClient.Close()
	}

	log.Info().Msg("Remnawave Node stopped")
}

// setupLogging configures zerolog
func setupLogging() {
	// Configure zerolog
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05.000"

	// Set log level based on environment
	if config.IsDevelopment() {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05.000",
		})
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05.000",
			NoColor:    true,
		})
	}
}
