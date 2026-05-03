package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/remnawave/remnanode/internal/config"
	"github.com/remnawave/remnanode/internal/server/middleware"
)

const bodyLimitBytes = 1 << 30 // 1 GiB, matching TS bodyParser.json({ limit: '1000mb' })

// Server represents the dual HTTP server setup
type Server struct {
	mainServer     *http.Server
	internalServer *http.Server
	mainRouter     *gin.Engine
	internalRouter *gin.Engine
	config         *config.Config
	services       *Services
}

// Services holds all service dependencies for handlers
type Services struct {
	// Will be populated with handler, stats, xray, vision, internal services
	Handler     interface{}
	Stats       interface{}
	Xray        interface{}
	Vision      interface{}
	Plugin      interface{}
	InternalAPI interface{}
}

// New creates a new Server instance
func New(cfg *config.Config, services *Services) (*Server, error) {
	// Set Gin mode based on environment
	if !config.IsDevelopment() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create main router (HTTPS with mTLS)
	mainRouter := gin.New()
	mainRouter.Use(middleware.Recovery())
	mainRouter.Use(middleware.SecureHeaders())
	mainRouter.Use(middleware.BodyLimit(bodyLimitBytes))
	mainRouter.Use(middleware.GzipDecompress()) // Decompress gzip request bodies
	mainRouter.Use(middleware.GzipCompress())   // Compress gzip response bodies (matching TS compression())
	mainRouter.Use(middleware.APIDiagnostics(cfg.Debug))
	if config.IsDevelopment() {
		mainRouter.Use(middleware.Logger())
	}
	// Do not trust X-Forwarded-For to prevent IP spoofing in ClientIP().
	if err := mainRouter.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("failed to set trusted proxies: %w", err)
	}

	// Create internal router (HTTP on localhost)
	internalRouter := gin.New()
	internalRouter.Use(middleware.Recovery())
	internalRouter.Use(middleware.BodyLimit(bodyLimitBytes))
	internalRouter.Use(middleware.InternalOnly())
	internalRouter.Use(middleware.TokenAuth(cfg.InternalRestToken))
	internalRouter.Use(middleware.APIDiagnostics(cfg.Debug))
	// Internal router also ignores proxy headers — it only accepts localhost connections.
	if err := internalRouter.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("failed to set trusted proxies on internal router: %w", err)
	}

	server := &Server{
		mainRouter:     mainRouter,
		internalRouter: internalRouter,
		config:         cfg,
		services:       services,
	}

	// Setup routes
	server.setupRoutes()

	// Close connection on unmatched routes (TS behavior)
	mainRouter.NoRoute(func(c *gin.Context) { middleware.CloseUnknownRoute(c) })
	internalRouter.NoRoute(func(c *gin.Context) { middleware.CloseUnknownRoute(c) })

	return server, nil
}

// Start starts both HTTP servers
func (s *Server) Start() error {
	// Get TLS configuration for main server
	tlsConfig, err := s.config.GetTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to create TLS config: %w", err)
	}

	// Create main HTTPS server (WriteTimeout=0 to avoid truncating long stats responses)
	s.mainServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.config.NodePort),
		Handler:           s.mainRouter,
		TLSConfig:         tlsConfig,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Create internal HTTP server
	s.internalServer = &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", config.XrayInternalAPIPort),
		Handler:           s.internalRouter,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start main HTTPS server in goroutine
	go func() {
		log.Info().
			Int("port", s.config.NodePort).
			Msg("Starting main HTTPS server with mTLS")

		// ListenAndServeTLS with empty strings uses certs from TLSConfig
		if err := s.mainServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Main server failed")
		}
	}()

	// Start internal HTTP server in goroutine
	go func() {
		log.Info().
			Int("port", config.XrayInternalAPIPort).
			Msg("Starting internal HTTP server")

		if err := s.internalServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Internal server failed")
		}
	}()

	return nil
}

// Shutdown gracefully shuts down both servers
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("Shutting down servers...")

	// Shutdown main server
	if err := s.mainServer.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Main server shutdown error")
	}

	// Shutdown internal server
	if err := s.internalServer.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Internal server shutdown error")
	}

	log.Info().Msg("Servers shutdown complete")
	return nil
}

// GetMainRouter returns the main router for adding routes
func (s *Server) GetMainRouter() *gin.Engine {
	return s.mainRouter
}

// GetInternalRouter returns the internal router for adding routes
func (s *Server) GetInternalRouter() *gin.Engine {
	return s.internalRouter
}
