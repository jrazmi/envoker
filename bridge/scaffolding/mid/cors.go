// Package mid contains middleware factory functions
package mid

import (
	"context"
	"net/http"

	"github.com/jrazmi/envoker/infrastructure/web"
)

// CORSConfig holds CORS configuration options
type CORSConfig struct {
	Origins     []string
	Methods     []string
	Headers     []string
	Credentials bool
	MaxAge      string
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		Origins:     []string{"*"},
		Methods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		Headers:     []string{"Accept", "Content-Type", "X-Token", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"},
		Credentials: true,
		MaxAge:      "86400",
	}
}

// CORS creates CORS middleware with the given origins
func CORS(origins ...string) web.MidFunc {
	config := DefaultCORSConfig()
	config.Origins = origins
	return CORSWithConfig(config)
}

// CORSWithConfig creates CORS middleware with full configuration
func CORSWithConfig(config CORSConfig) web.MidFunc {
	return func(handler web.HandlerFunc) web.HandlerFunc {
		return func(ctx context.Context, r *http.Request) web.Encoder {
			w := web.GetWriter(ctx)

			reqOrigin := r.Header.Get("Origin")

			// Set allowed origin
			for _, origin := range config.Origins {
				if origin == "*" || origin == reqOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}

			// Set credentials
			if config.Credentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Set allowed methods
			if len(config.Methods) > 0 {
				methods := ""
				for i, method := range config.Methods {
					if i > 0 {
						methods += ", "
					}
					methods += method
				}
				w.Header().Set("Access-Control-Allow-Methods", methods)
			}

			// Set allowed headers
			if len(config.Headers) > 0 {
				headers := ""
				for i, header := range config.Headers {
					if i > 0 {
						headers += ", "
					}
					headers += header
				}
				w.Header().Set("Access-Control-Allow-Headers", headers)
			}

			// Set max age
			if config.MaxAge != "" {
				w.Header().Set("Access-Control-Max-Age", config.MaxAge)
			}

			return handler(ctx, r)
		}
	}
}

// AdminCORS creates CORS middleware for admin routes (more restrictive)
func AdminCORS(adminDomains ...string) web.MidFunc {
	config := CORSConfig{
		Origins:     adminDomains,
		Methods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		Headers:     []string{"Accept", "Content-Type", "Authorization", "X-CSRF-Token"},
		Credentials: true,
		MaxAge:      "3600", // Shorter cache for admin
	}
	return CORSWithConfig(config)
}

// APICORS creates CORS middleware for API routes
func APICORS(apiOrigins ...string) web.MidFunc {
	config := CORSConfig{
		Origins:     apiOrigins,
		Methods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		Headers:     []string{"Accept", "Content-Type", "Authorization", "X-API-Key"},
		Credentials: false, // APIs often don't need credentials
		MaxAge:      "86400",
	}
	return CORSWithConfig(config)
}

// PublicCORS creates CORS middleware for public routes (most permissive)
func PublicCORS() web.MidFunc {
	return CORS("*")
}
