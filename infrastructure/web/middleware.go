package web

import (
	"context"
	"net/http"
)

// ============================================================================
// Helper Methods
// ============================================================================

func (wh *WebHandler) buildHandlerChain(handler HandlerFunc, middleware ...Middleware) HandlerFunc {
	allMiddleware := append(wh.globalMiddleware, middleware...)

	final := handler
	for i := len(allMiddleware) - 1; i >= 0; i-- {
		final = allMiddleware[i](final)
	}

	return final
}

func (wh *WebHandler) corsMiddleware() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, r *http.Request) Encoder {
			// Get writer from context - should now be available
			w := GetWriter(ctx)

			// Add safety check
			if w == nil {
				// This should not happen if handlers.go is fixed, but safety first
				return NewError("internal server error: response writer not available")
			}

			origin := r.Header.Get("Origin")
			for _, allowedOrigin := range wh.corsOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
					break
				}
			}

			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == "OPTIONS" {
				return NewNoResponse()
			}

			return next(ctx, r)
		}
	}
}
