package web

import "strings"

// MiddlewareAction defines how middleware should be applied
type MiddlewareAction int

const (
	MiddlewareAdd     MiddlewareAction = iota // Add middleware
	MiddlewareReplace                         // Replace specific middleware
	MiddlewareSkip                            // Skip specific middleware
)

// MidFunc is a handler function designed to run code before and/or after
// another Handler. It is designed to remove boilerplate or other concerns not
// direct to any given app Handler.
type MidFunc func(handler HandlerFunc) HandlerFunc

// MiddlewareOverride allows path groups to modify global middleware behavior
type MiddlewareOverride struct {
	Action     MiddlewareAction
	Target     string    // Name of global middleware to target
	Middleware []MidFunc // Middleware to add or use as replacement
}

// PathGroupMiddleware holds middleware for a specific URL path group
type PathGroupMiddleware struct {
	Pattern    string               // URL pattern to match (e.g., "/admin")
	Middleware []MidFunc            // Path group-specific middleware
	Overrides  []MiddlewareOverride // How to handle global middleware
}

// AddGlobalMiddleware adds named global middleware
func (a *App) AddGlobalMiddleware(name string, mw MidFunc) {
	a.globalMw[name] = mw
}

// AddPathGroupMiddleware adds middleware for a specific URL pattern
func (a *App) AddPathGroupMiddleware(pattern string, mw ...MidFunc) {
	a.pathGroupMw = append(a.pathGroupMw, PathGroupMiddleware{
		Pattern:    pattern,
		Middleware: mw,
	})
}

// AddPathGroupMiddlewareWithOverrides adds path group middleware with global middleware overrides
func (a *App) AddPathGroupMiddlewareWithOverrides(pattern string, overrides []MiddlewareOverride, mw ...MidFunc) {
	a.pathGroupMw = append(a.pathGroupMw, PathGroupMiddleware{
		Pattern:    pattern,
		Middleware: mw,
		Overrides:  overrides,
	})
}

// getPathMWConfig returns path group middleware and overrides for a given path
func (a *App) getPathMWConfig(path string) ([]MidFunc, []MiddlewareOverride) {
	for _, pathGroup := range a.pathGroupMw {
		if strings.HasPrefix(path, pathGroup.Pattern) {
			return pathGroup.Middleware, pathGroup.Overrides
		}
	}
	return nil, nil // No path group-specific config
}

// buildMiddlewareChain constructs the final middleware chain
func (a *App) buildMiddlewareChain(pathGroupMw []MidFunc, overrides []MiddlewareOverride, routeMw []MidFunc) []MidFunc {
	var finalChain []MidFunc

	// 1. Route-specific middleware first
	finalChain = append(finalChain, routeMw...)

	// 2. Path group-specific middleware
	finalChain = append(finalChain, pathGroupMw...)

	// 3. Process global middleware with overrides
	skipNames := make(map[string]bool)
	replaceNames := make(map[string][]MidFunc)

	// Process overrides to determine what to skip/replace
	for _, override := range overrides {
		switch override.Action {
		case MiddlewareSkip:
			skipNames[override.Target] = true
		case MiddlewareReplace:
			replaceNames[override.Target] = override.Middleware
		case MiddlewareAdd:
			finalChain = append(finalChain, override.Middleware...)
		}
	}

	// Add global middleware (in a consistent order)
	globalOrder := a.mwGlobalOrder

	for _, name := range globalOrder {
		if mw, exists := a.globalMw[name]; exists {
			if skipNames[name] {
				continue // Skip this middleware
			}
			if replacement, shouldReplace := replaceNames[name]; shouldReplace {
				finalChain = append(finalChain, replacement...) // Use replacement
			} else {
				finalChain = append(finalChain, mw) // Use original
			}
		}
	}

	// Add any global middleware not in the standard order
	for name, mw := range a.globalMw {
		found := false
		for _, orderedName := range globalOrder {
			if name == orderedName {
				found = true
				break
			}
		}
		if !found {
			if !skipNames[name] {
				if replacement, shouldReplace := replaceNames[name]; shouldReplace {
					finalChain = append(finalChain, replacement...)
				} else {
					finalChain = append(finalChain, mw)
				}
			}
		}
	}

	return finalChain
}

// getCORSMiddleware returns the appropriate CORS middleware considering overrides
func (a *App) getCORSMiddleware(overrides []MiddlewareOverride) MidFunc {
	// Check if CORS is being overridden
	for _, override := range overrides {
		if override.Target == "cors" {
			switch override.Action {
			case MiddlewareSkip:
				return nil // No CORS
			case MiddlewareReplace:
				if len(override.Middleware) > 0 {
					return override.Middleware[0] // Use first replacement CORS middleware
				}
			}
		}
	}

	// Return default global CORS middleware if it exists
	if corsMiddleware, exists := a.globalMw["cors"]; exists {
		return corsMiddleware
	}

	return nil // No CORS middleware
}

// wrapMiddleware creates a new handler by wrapping middleware around a final
// handler. The middlewares' Handlers will be executed by requests in the order
// they are provided.
func wrapMiddleware(mw []MidFunc, handler HandlerFunc) HandlerFunc {

	// Loop backwards through the middleware invoking each one. Replace the
	// handler with the new wrapped handler. Looping backwards ensures that the
	// first middleware of the slice is the first to be executed by requests.
	for i := len(mw) - 1; i >= 0; i-- {
		mwFunc := mw[i]
		if mwFunc != nil {
			handler = mwFunc(handler)
		}
	}

	return handler
}
