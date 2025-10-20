// Package usersessionsrepobridge contains HTTP route registration for UserSession
// This file is generated once and can be customized.
// It will NOT be overwritten by the generator.

package usersessionsrepobridge

import (
	"github.com/jrazmi/envoker/core/repositories/usersessionsrepo"
	"github.com/jrazmi/envoker/infrastructure/web"
	"github.com/jrazmi/envoker/sdk/logger"
)

// Config holds configuration for the UserSession bridge
type Config struct {
	Log        *logger.Logger
	Repository *usersessionsrepo.Repository
	Middleware []web.Middleware
}

// AddHttpRoutes registers all HTTP routes for UserSession
// See http_gen.go for available handler methods and suggested routes
func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
	b := newBridge(cfg.Repository)

	// Standard CRUD routes
	group.GET("/user-sessions", b.httpList)
	group.GET("/user-sessions/{session_id}", b.httpGetByID)
	group.POST("/user-sessions", b.httpCreate)
	group.PUT("/user-sessions/{session_id}", b.httpUpdate)
	group.DELETE("/user-sessions/{session_id}", b.httpDelete)

	// Foreign key routes
	group.GET("/users/{user_id}/user-sessions", b.httpListByUserId)
}
