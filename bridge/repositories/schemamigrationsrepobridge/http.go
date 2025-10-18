// Package schemamigrationsrepobridge contains HTTP route registration for SchemaMigration
// This file is generated once and can be customized.
// It will NOT be overwritten by the generator.

package schemamigrationsrepobridge

import (
	"github.com/jrazmi/envoker/core/repositories/schemamigrationsrepo"
	"github.com/jrazmi/envoker/infrastructure/web"
	"github.com/jrazmi/envoker/sdk/logger"
)

// Config holds configuration for the SchemaMigration bridge
type Config struct {
	Log        *logger.Logger
	Repository *schemamigrationsrepo.Repository
	Middleware []web.Middleware
}

// AddHttpRoutes registers all HTTP routes for SchemaMigration
// See http_gen.go for available handler methods and suggested routes
func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
	b := newBridge(cfg.Repository)

	// Standard CRUD routes
	group.GET("/schema-migrations", b.httpList)
	group.GET("/schema-migrations/{version}", b.httpGetByID)
	group.POST("/schema-migrations", b.httpCreate)
	group.PUT("/schema-migrations/{version}", b.httpUpdate)
	group.DELETE("/schema-migrations/{version}", b.httpDelete)
}
