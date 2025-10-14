package web

import "strings"

type RouteGroup struct {
	webHandler *WebHandler
	prefix     string
	middleware []Middleware
}

func (wh *WebHandler) Group(prefix string, middleware ...Middleware) *RouteGroup {
	return &RouteGroup{
		webHandler: wh,
		prefix:     strings.TrimSuffix(prefix, "/"),
		middleware: middleware,
	}
}

func (g *RouteGroup) Handle(method, path string, handler HandlerFunc, middleware ...Middleware) {
	allMiddleware := append(g.middleware, middleware...)
	fullPath := g.prefix + path
	g.webHandler.Handle(method, fullPath, handler, allMiddleware...)
}

func (g *RouteGroup) Group(prefix string, middleware ...Middleware) *RouteGroup {
	combinedMiddleware := append(g.middleware, middleware...)
	return &RouteGroup{
		webHandler: g.webHandler,
		prefix:     g.prefix + strings.TrimSuffix(prefix, "/"),
		middleware: combinedMiddleware,
	}
}
