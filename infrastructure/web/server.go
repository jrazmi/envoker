package web

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jrazmi/envoker/sdk/environment"
)

// WebServer wraps http.Server with additional configuration
type WebServer struct {
	*http.Server
	Config ServerConfig
}

// ServerConfig holds web server configuration (exportable)
type ServerConfig struct {
	Port            string        `toml:"port" env:"PORT" default:":8080"`
	EnableDebug     bool          `toml:"enable_debug" env:"ENABLE_DEBUG" default:"false"`
	ReadTimeout     time.Duration `toml:"read_timeout" env:"READ_TIMEOUT" default:"30s"`
	WriteTimeout    time.Duration `toml:"write_timeout" env:"WRITE_TIMEOUT" default:"10s"`
	IdleTimeout     time.Duration `toml:"idle_timeout" env:"IDLE_TIMEOUT" default:"120s"`
	ShutdownTimeout time.Duration `toml:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT" default:"20s"`
}

// internal serveroptions struct for runtime configuration
type serveroptions struct {
	handler  http.Handler
	errorLog *log.Logger
	config   ServerConfig
}

// ServerOption takes config serveroption and returns formatted config
type ServerOption func(*serveroptions)

// WithHandler sets the HTTP handler
func WithHandler(handler http.Handler) ServerOption {
	return func(o *serveroptions) {
		o.handler = handler
	}
}

// WithErrorLog sets the error logger
func WithErrorLog(errorLog *log.Logger) ServerOption {
	return func(o *serveroptions) {
		o.errorLog = errorLog
	}
}

// WithPort sets the server port
func WithPort(port string) ServerOption {
	return func(o *serveroptions) {
		o.config.Port = port
	}
}

// WithTimeouts sets all timeout values
func WithTimeouts(read, write, idle, shutdown time.Duration) ServerOption {
	return func(o *serveroptions) {
		o.config.ReadTimeout = read
		o.config.WriteTimeout = write
		o.config.IdleTimeout = idle
		o.config.ShutdownTimeout = shutdown
	}
}

// WithReadTimeout sets the read timeout
func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(o *serveroptions) {
		o.config.ReadTimeout = timeout
	}
}

// WithWriteTimeout sets the write timeout
func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(o *serveroptions) {
		o.config.WriteTimeout = timeout
	}
}

// WithIdleTimeout sets the idle timeout
func WithIdleTimeout(timeout time.Duration) ServerOption {
	return func(o *serveroptions) {
		o.config.IdleTimeout = timeout
	}
}

// WithShutdownTimeout sets the shutdown timeout
func WithShutdownTimeout(timeout time.Duration) ServerOption {
	return func(o *serveroptions) {
		o.config.ShutdownTimeout = timeout
	}
}

// WithDebug enables debug mode
func WithDebug(enabled bool) ServerOption {
	return func(o *serveroptions) {
		o.config.EnableDebug = enabled
	}
}

// ============================================================================
// Exported Constructor Functions
// ============================================================================

// NewDefault creates a new WebServer with default settings
func NewServerDefault(opts ...ServerOption) *WebServer {
	config := ServerConfig{
		Port:            ":8080",
		EnableDebug:     false,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 20 * time.Second,
	}
	return newWebServer(config, opts...)
}

// NewFromEnv creates a new WebServer from environment variables
func NewServerFromEnv(prefix string, opts ...ServerOption) (*WebServer, error) {
	var config ServerConfig
	if err := environment.ParseEnvTags(prefix, &config); err != nil {
		return nil, fmt.Errorf("parsing webserver config: %w", err)
	}

	return newWebServer(config, opts...), nil
}

// ============================================================================
// Internal Constructor
// ============================================================================

// newWebServer creates a new WebServer with given config and applies serveroptions
func newWebServer(cfg ServerConfig, opts ...ServerOption) *WebServer {
	// Start with config-based serveroptions
	internalOpts := &serveroptions{
		config: cfg,
	}

	// Apply functional serveroptions
	for _, opt := range opts {
		opt(internalOpts)
	}

	// Create the underlying http.Server
	server := &http.Server{
		Addr:         internalOpts.config.Port,
		Handler:      internalOpts.handler,
		ReadTimeout:  internalOpts.config.ReadTimeout,
		WriteTimeout: internalOpts.config.WriteTimeout,
		IdleTimeout:  internalOpts.config.IdleTimeout,
		ErrorLog:     internalOpts.errorLog,
	}

	return &WebServer{
		Server: server,
		Config: internalOpts.config,
	}
}
