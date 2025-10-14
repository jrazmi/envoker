package web

func (wh *WebHandler) GET(path string, handler HandlerFunc, middleware ...Middleware) {
	wh.Handle("GET", path, handler, middleware...)
}

func (wh *WebHandler) POST(path string, handler HandlerFunc, middleware ...Middleware) {
	wh.Handle("POST", path, handler, middleware...)
}

func (wh *WebHandler) PUT(path string, handler HandlerFunc, middleware ...Middleware) {
	wh.Handle("PUT", path, handler, middleware...)
}

func (wh *WebHandler) DELETE(path string, handler HandlerFunc, middleware ...Middleware) {
	wh.Handle("DELETE", path, handler, middleware...)
}

func (g *RouteGroup) GET(path string, handler HandlerFunc, middleware ...Middleware) {
	g.Handle("GET", path, handler, middleware...)
}

func (g *RouteGroup) POST(path string, handler HandlerFunc, middleware ...Middleware) {
	g.Handle("POST", path, handler, middleware...)
}

func (g *RouteGroup) PUT(path string, handler HandlerFunc, middleware ...Middleware) {
	g.Handle("PUT", path, handler, middleware...)
}

func (g *RouteGroup) DELETE(path string, handler HandlerFunc, middleware ...Middleware) {
	g.Handle("DELETE", path, handler, middleware...)
}
