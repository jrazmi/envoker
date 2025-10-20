// Package usersrepobridge contains HTTP route registration for User
// This file is generated once and can be customized.
// It will NOT be overwritten by the generator.

package usersrepobridge

import (
	"github.com/jrazmi/envoker/core/repositories/usersrepo"
	"github.com/jrazmi/envoker/infrastructure/web"
	"github.com/jrazmi/envoker/sdk/logger"
)

// Config holds configuration for the User bridge
type Config struct {
	Log        *logger.Logger
	Repository *usersrepo.Repository
	Middleware []web.Middleware
}

// AddHttpRoutes registers all HTTP routes for User
// See http_gen.go for available handler methods and suggested routes
func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
	b := newBridge(cfg.Repository)

	// Standard CRUD routes
	group.GET("/users", b.httpList)
	group.GET("/users/{user_id}", b.httpGetByID)
	group.POST("/users", b.httpCreate)
	group.PUT("/users/{user_id}", b.httpUpdate)
	group.DELETE("/users/{user_id}", b.httpDelete)
}
