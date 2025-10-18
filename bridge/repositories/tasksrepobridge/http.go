// Package tasksrepobridge contains HTTP route registration for Task
// This file is generated once and can be customized.
// It will NOT be overwritten by the generator.

package tasksrepobridge

import (
	"github.com/jrazmi/envoker/core/repositories/tasksrepo"
	"github.com/jrazmi/envoker/infrastructure/web"
	"github.com/jrazmi/envoker/sdk/logger"
)

// Config holds configuration for the Task bridge
type Config struct {
	Log        *logger.Logger
	Repository *tasksrepo.Repository
	Middleware []web.Middleware
}

// AddHttpRoutes registers all HTTP routes for Task
// See http_gen.go for available handler methods and suggested routes
func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
	b := newBridge(cfg.Repository)

	// Standard CRUD routes
	group.GET("/tasks", b.httpList)
	group.GET("/tasks/{task_id}", b.httpGetByID)
	group.POST("/tasks", b.httpCreate)
	group.PUT("/tasks/{task_id}", b.httpUpdate)
	group.DELETE("/tasks/{task_id}", b.httpDelete)
}
