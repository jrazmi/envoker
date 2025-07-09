// Package web contains a small web framework extension.
package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jrazmi/envoker/sdk/logger"
)

// Encoder defines behavior that can encode a data model and provide
// the content type for that encoding.
type Encoder interface {
	Encode() (data []byte, contentType string, err error)
}

// HandlerFunc represents a function that handles a http request within our own
// little mini framework.
type HandlerFunc func(ctx context.Context, r *http.Request) Encoder

// Telemetry represents a function that can call telemetry functions
type Telemetry interface {
	SetTraceID(ctx context.Context) context.Context
	GetTraceID(ctx context.Context) string
}

// App is the entrypoint into our application and what configures our context
// object for each of our http handlers.
type App struct {
	log           *logger.Logger
	mux           *http.ServeMux
	telemetry     Telemetry
	globalMw      map[string]MidFunc    // All global middleware by name
	pathGroupMw   []PathGroupMiddleware // Path group-specific middleware
	mwGlobalOrder []string
}

// NewApp creates an App value that handle a set of routes for the application.
func NewApp(log *logger.Logger, telemetry Telemetry) *App {
	mux := http.NewServeMux()
	return &App{
		log:           log,
		telemetry:     telemetry,
		mux:           mux,
		globalMw:      make(map[string]MidFunc),
		pathGroupMw:   make([]PathGroupMiddleware, 0),
		mwGlobalOrder: globalMWOrder,
	}
}

// ServeHTTP implements the http.Handler interface.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

// HandlerFuncNoMid sets a handler function without any middleware
func (a *App) HandlerFuncNoMid(method string, group string, path string, handlerFunc HandlerFunc) {
	h := func(w http.ResponseWriter, r *http.Request) {
		ctx := a.telemetry.SetTraceID(r.Context())
		ctx = setWriter(ctx, w)
		resp := handlerFunc(ctx, r)

		if err := Respond(ctx, w, resp); err != nil {
			a.log.Error(ctx, "web-respond", "err", err)
			return
		}
	}

	finalPath := a.buildFinalPath(method, group, path)
	a.mux.HandleFunc(finalPath, h)
}

// HandlerFunc sets a handler function with middleware applied
func (a *App) HandlerFunc(method string, group string, path string, handlerFunc HandlerFunc, mw ...MidFunc) {
	// Build the full path to determine section middleware
	routePath := path
	if group != "" {
		routePath = "/" + group + path
	}

	// Get path-specific middleware and overrides
	pathMw, overrides := a.getPathMWConfig(routePath)

	// Build the complete middleware chain
	middlewareChain := a.buildMiddlewareChain(pathMw, overrides, mw)

	// Handle CORS separately - it should run first (outermost)
	corsMiddleware := a.getCORSMiddleware(overrides)
	if corsMiddleware != nil {
		handlerFunc = corsMiddleware(handlerFunc)
	}

	// Apply all middleware (including CORS if it's in the global middleware)
	handlerFunc = wrapMiddleware(middlewareChain, handlerFunc)

	h := func(w http.ResponseWriter, r *http.Request) {
		ctx := a.telemetry.SetTraceID(r.Context())
		ctx = setWriter(ctx, w)

		resp := handlerFunc(ctx, r)

		if err := Respond(ctx, w, resp); err != nil {
			a.log.Error(ctx, "web-respond", "err", err)
			return
		}
	}

	finalPath := a.buildFinalPath(method, group, path)
	a.mux.HandleFunc(finalPath, h)
}

// buildFinalPath constructs the final path for the mux
func (a *App) buildFinalPath(method, group, path string) string {
	finalPath := path
	if group != "" {
		finalPath = "/" + group + path
	}
	return fmt.Sprintf("%s %s", method, finalPath)
}
