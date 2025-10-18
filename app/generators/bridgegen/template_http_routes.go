package bridgegen

const HTTPRoutesTemplate = `// Package {{.PackageName}} contains HTTP route registration for {{.EntityName}}
// This file is generated once and can be customized.
// It will NOT be overwritten by the generator.

package {{.PackageName}}

import (
	"{{.ModulePath}}/core/repositories/{{.RepoPackage}}"
	"{{.ModulePath}}/infrastructure/web"
	"{{.ModulePath}}/sdk/logger"
)

// Config holds configuration for the {{.EntityName}} bridge
type Config struct {
	Log        *logger.Logger
	Repository *{{.RepoPackage}}.Repository
	Middleware []web.Middleware
}

// AddHttpRoutes registers all HTTP routes for {{.EntityName}}
// See http_gen.go for available handler methods and suggested routes
func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
	b := newBridge(cfg.Repository)

	// Standard CRUD routes
	group.GET("{{.HTTPBasePath}}", b.httpList)
	group.GET("{{printf "%s/{%s}" .HTTPBasePath .PKURLParam}}", b.httpGetByID)
	group.POST("{{.HTTPBasePath}}", b.httpCreate)
	group.PUT("{{printf "%s/{%s}" .HTTPBasePath .PKURLParam}}", b.httpUpdate)
	group.DELETE("{{printf "%s/{%s}" .HTTPBasePath .PKURLParam}}", b.httpDelete)
{{- if .ForeignKeys}}

	// Foreign key routes
{{- range .ForeignKeys}}
	group.GET("{{.RoutePath}}", b.{{.MethodName}})
{{- end}}
{{- end}}
}
`
