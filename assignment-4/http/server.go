package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/validate"
	"github.com/rs/cors"
	"github.com/stupside/DATA-intensive/assignment-4/auth"
	"github.com/stupside/DATA-intensive/assignment-4/config"
	"github.com/stupside/DATA-intensive/assignment-4/device"
	"github.com/stupside/DATA-intensive/assignment-4/internal/repository"
	"github.com/stupside/DATA-intensive/assignment-4/proto/gen/spec/v1/v1connect"
	"github.com/stupside/DATA-intensive/assignment-4/relay"
	"github.com/stupside/DATA-intensive/assignment-4/share"
)

// Server represents the HTTP server with all dependencies
type Server struct {
	cfg          *config.Config
	httpServer   *http.Server
	sessionStore *relay.SessionStore

	// Repository fields organized by database type
	dbRepos *repository.CompositeRepositories
}

// New creates a new server instance
func New(cfg *config.Config, repos *repository.CompositeRepositories) (*Server, error) {
	s := &Server{
		cfg:     cfg,
		dbRepos: repos,
	}

	// Initialize Postgres and Mongo repositories and merge them into a driver-agnostic facade
	// `repos` must be built by the caller; just assign in constructor

	// Initialize session store with session repository
	s.sessionStore = relay.NewSessionStore()

	return s, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Setup handler chain
	handler, err := s.buildHandlerChain()
	if err != nil {
		return err
	}

	// Configure TLS
	tlsConfig, err := s.buildTLSConfig()
	if err != nil {
		return err
	}

	// Create HTTP server
	// Note: HTTP/2 is automatically enabled for TLS connections when NextProtos includes "h2"
	s.httpServer = &http.Server{
		Addr:      fmt.Sprintf(":%d", s.cfg.Server.Port),
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	// Start listening
	return s.httpServer.ListenAndServeTLS(
		s.cfg.TLS.Server.CertPath,
		s.cfg.TLS.Server.KeyPath,
	)
}

// buildHandlerChain builds the complete handler chain with middleware
func (s *Server) buildHandlerChain() (http.Handler, error) {
	// Create mux and register routes
	mux := http.NewServeMux()
	if err := s.registerRoutes(mux); err != nil {
		return nil, err
	}

	// Apply middleware: CORS -> mTLS -> mux
	return s.applyCORS(s.applyMTLS(mux)), nil
}

// registerRoutes registers all service routes
func (s *Server) registerRoutes(mux *http.ServeMux) error {
	// Register gRPC reflection if enabled
	if s.cfg.Features.GRPC.EnableReflection {
		s.registerReflection(mux)
	}

	// Register services
	s.registerRelayService(mux)
	s.registerShareService(mux)
	if err := s.registerAuthService(mux); err != nil {
		return err
	}
	s.registerDeviceService(mux)

	return nil
}

// registerReflection registers gRPC reflection handler
func (s *Server) registerReflection(mux *http.ServeMux) {
	reflector := grpcreflect.NewStaticReflector(
		v1connect.AuthServiceName,
		v1connect.DeviceServiceName,
		v1connect.RelayServiceName,
		v1connect.ShareServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
}

// registerRelayService registers the relay (streaming) service
func (s *Server) registerRelayService(mux *http.ServeMux) {
	// Create relay handler with all repository dependencies
	repos := &relay.Repositories{
		Share:    s.dbRepos.Share,
		Relay:    s.dbRepos.Relay,
		Streamer: s.dbRepos.Streamer,
		Consumer: s.dbRepos.Consumer,
		Chunk:    s.dbRepos.Chunk,
	}
	relayHandler := relay.NewHandler(
		s.cfg,
		repos,
		s.sessionStore,
	)

	path, handler := v1connect.NewRelayServiceHandler(
		relayHandler,
		connect.WithInterceptors(
			validate.NewInterceptor(),
			auth.AuthInterceptor(s.cfg, s.dbRepos.Device),
		),
	)
	mux.Handle(path, handler)
}

// registerShareService registers the share management service
func (s *Server) registerShareService(mux *http.ServeMux) {
	shareService := share.NewService(s.dbRepos.Share)

	path, handler := v1connect.NewShareServiceHandler(
		shareService,
		connect.WithInterceptors(
			validate.NewInterceptor(),
			auth.AuthInterceptor(s.cfg, s.dbRepos.Device),
		),
	)
	mux.Handle(path, handler)
}

// registerAuthService registers the device onboarding service
func (s *Server) registerAuthService(mux *http.ServeMux) error {
	// Initialize auth service
	authService, err := auth.NewService(s.cfg, s.dbRepos.Device)
	if err != nil {
		return fmt.Errorf("failed to create auth service: %w", err)
	}

	path, handler := v1connect.NewAuthServiceHandler(
		authService,
		connect.WithInterceptors(validate.NewInterceptor()),
	)
	mux.Handle(path, handler)

	return nil
}

// registerDeviceService registers the device information service
func (s *Server) registerDeviceService(mux *http.ServeMux) {
	// Initialize device service
	deviceService := device.NewService(s.dbRepos.Connection)

	path, handler := v1connect.NewDeviceServiceHandler(
		deviceService,
		connect.WithInterceptors(
			validate.NewInterceptor(),
			auth.AuthInterceptor(s.cfg, s.dbRepos.Device),
		),
	)
	mux.Handle(path, handler)
}

// applyMTLS applies the mTLS middleware
func (s *Server) applyMTLS(handler http.Handler) http.Handler {
	return auth.MTLSInterceptor(s.cfg, s.dbRepos.Certificate, s.dbRepos.Connection)(handler)
}

// applyCORS applies the CORS middleware
func (s *Server) applyCORS(handler http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins:   s.cfg.Security.CORS.AllowedOrigins,
		AllowedMethods:   connectcors.AllowedMethods(),
		AllowedHeaders:   connectcors.AllowedHeaders(),
		ExposedHeaders:   connectcors.ExposedHeaders(),
		AllowCredentials: true,
	}).Handler(handler)
}

// buildTLSConfig builds the TLS configuration
func (s *Server) buildTLSConfig() (*tls.Config, error) {
	caCertPool, err := s.loadCACertPool()
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		ClientAuth: tls.VerifyClientCertIfGiven, // Optional client cert - verified if provided
		ClientCAs:  caCertPool,
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"}, // Enable HTTP/2 and HTTP/1.1 via ALPN
	}, nil
}

// loadCACertPool loads the CA certificate pool for client verification
func (s *Server) loadCACertPool() (*x509.CertPool, error) {
	caCertPEM, err := os.ReadFile(s.cfg.TLS.CA.CertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return caCertPool, nil
}
