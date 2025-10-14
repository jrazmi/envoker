package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jrazmi/envoker/sdk/environment"
)

// HandlerFunc represents a function that handles a http request and returns something to encode
type HandlerFunc func(ctx context.Context, r *http.Request) Encoder

// Middleware wraps a HandlerFunc
type Middleware func(HandlerFunc) HandlerFunc

type Telemetry interface {
	SetTraceID(ctx context.Context) context.Context
	GetTraceID(ctx context.Context) string
}

type WebHandler struct {
	mux       *http.ServeMux
	log       *slog.Logger
	telemetry Telemetry

	// Configuration
	corsOrigins    []string
	defaultHeaders map[string]string

	// Middleware stacks
	globalMiddleware []Middleware
}

// Options is the exportable configuration struct
type HandlerOptions struct {
	CORSOrigins    []string          `yaml:"cors_origins" toml:"cors_origins" json:"cors_origins" env:"CORS_ORIGINS" default:"*" separator:","`
	DefaultHeaders map[string]string `yaml:"default_headers" toml:"default_headers" json:"default_headers"`
}

type HandlerOption func(*handlerOptions)

// internal options struct for additional runtime configuration
type handlerOptions struct {
	log              *slog.Logger
	telemetry        Telemetry
	corsOrigins      []string
	defaultHeaders   map[string]string
	globalMiddleware []Middleware
}

// WithLogging sets the logger
func WithLogging(log *slog.Logger) HandlerOption {
	return func(o *handlerOptions) {
		o.log = log
	}
}

// WithTelemetry sets the telemetry provider
func WithTelemetry(tel Telemetry) HandlerOption {
	return func(o *handlerOptions) {
		o.telemetry = tel
	}
}

// WithCORS sets CORS origins
func WithCORS(origins []string) HandlerOption {
	return func(o *handlerOptions) {
		o.corsOrigins = origins
	}
}

// WithDefaultHeaders sets default headers
func WithDefaultHeaders(headers map[string]string) HandlerOption {
	return func(o *handlerOptions) {
		if o.defaultHeaders == nil {
			o.defaultHeaders = make(map[string]string)
		}
		for k, v := range headers {
			o.defaultHeaders[k] = v
		}
	}
}

// WithGlobalMiddleware adds global middleware
func WithGlobalMiddleware(middleware ...Middleware) HandlerOption {
	return func(o *handlerOptions) {
		o.globalMiddleware = append(o.globalMiddleware, middleware...)
	}
}

// NewFromEnv creates a new WebHandler from environment variables
func NewWebHandlerFromEnv(prefix string, opts ...HandlerOption) (*WebHandler, error) {
	var options HandlerOptions
	if err := environment.ParseEnvTags(prefix, &options); err != nil {
		return nil, fmt.Errorf("parsing webhandler config: %w", err)
	}
	return newWebHandler(options, opts...), nil
}

// newWebHandler creates a new WebHandler with given config and applies options
func newWebHandler(cfg HandlerOptions, opts ...HandlerOption) *WebHandler {
	// Start with config-based options
	internalOpts := &handlerOptions{
		corsOrigins:      cfg.CORSOrigins,
		defaultHeaders:   cfg.DefaultHeaders,
		globalMiddleware: make([]Middleware, 0),
	}

	// Ensure defaultHeaders is initialized
	if internalOpts.defaultHeaders == nil {
		internalOpts.defaultHeaders = make(map[string]string)
	}

	// Apply functional options
	for _, opt := range opts {
		opt(internalOpts)
	}

	// Create the WebHandler
	handler := &WebHandler{
		mux:              http.NewServeMux(),
		log:              internalOpts.log,
		telemetry:        internalOpts.telemetry,
		corsOrigins:      internalOpts.corsOrigins,
		defaultHeaders:   internalOpts.defaultHeaders,
		globalMiddleware: internalOpts.globalMiddleware,
	}

	// After all options are applied, if CORS is configured, prepend it to the middleware chain
	// This ensures CORS runs first (before Logger, Errors, etc.)
	if len(handler.corsOrigins) > 0 {
		corsMiddleware := handler.corsMiddleware()
		// Prepend CORS to run before all other middleware
		handler.globalMiddleware = append([]Middleware{corsMiddleware}, handler.globalMiddleware...)
	}

	return handler
}

func (a *WebHandler) Handle(method, path string, handler HandlerFunc, middleware ...Middleware) {
	finalHandler := a.buildHandlerChain(handler, middleware...)

	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if a.telemetry != nil {
			ctx = a.telemetry.SetTraceID(ctx)
		}
		ctx = setWriter(ctx, w)
		// Set default headers
		for k, v := range a.defaultHeaders {
			w.Header().Set(k, v)
		}

		resp := finalHandler(ctx, r)

		if err := Respond(ctx, w, resp); err != nil && a.log != nil {
			a.log.ErrorContext(ctx, "respond error", "error", err)
		}
	}

	pattern := fmt.Sprintf("%s %s", strings.ToUpper(method), path)
	a.mux.HandleFunc(pattern, httpHandler)
}

// Raw handler registration (for when you need full control).  This does not apply global middleware.
func (a *WebHandler) HandleRaw(pattern string, handler http.Handler) {
	a.mux.Handle(pattern, handler)
}

func (a *WebHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}
