package pisigo

import "strings"

type Route struct {
	Method  string
	Path    string
	Handler HandlerFunc
	Name    string
	app     *App
}

func (r *Route) SetName(name string) *Route {
	r.Name = name
	if r.app != nil && name != "" {
		r.app.namedRoutes[name] = r
	}
	return r
}

type Group struct {
	app         *App
	prefix      string
	middlewares []Middleware
}

func (a *App) addRoute(method string, path string, handler HandlerFunc) *Route {
	route := Route{
		Method:  method,
		Path:    path,
		Handler: handler,
		app:     a,
	}
	a.routes = append(a.routes, route)
	return &a.routes[len(a.routes)-1]
}

func (a *App) Group(prefix string, middlewares ...Middleware) *Group {
	return &Group{
		app:         a,
		prefix:      prefix,
		middlewares: middlewares,
	}
}

func (g *Group) Use(middlewares ...Middleware) {
	g.middlewares = append(g.middlewares, middlewares...)
}

func (g *Group) Group(prefix string, middlewares ...Middleware) *Group {
	combined := append([]Middleware{}, g.middlewares...)
	combined = append(combined, middlewares...)
	return &Group{
		app:         g.app,
		prefix:      g.prefix + prefix,
		middlewares: combined,
	}
}

func (g *Group) addRoute(method string, path string, handler HandlerFunc) *Route {
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		handler = g.middlewares[i](handler)
	}
	return g.app.addRoute(method, joinRoutePath(g.prefix, path), handler)
}

func joinRoutePath(prefix, path string) string {
	if path == "" {
		return prefix
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if prefix == "" {
		return path
	}
	return strings.TrimRight(prefix, "/") + path
}

func (a *App) GET(path string, handler HandlerFunc) *Route {
	return a.addRoute(httpMethodGet, path, handler)
}

func (a *App) POST(path string, handler HandlerFunc) *Route {
	return a.addRoute(httpMethodPost, path, handler)
}

func (a *App) PUT(path string, handler HandlerFunc) *Route {
	return a.addRoute(httpMethodPut, path, handler)
}

func (a *App) PATCH(path string, handler HandlerFunc) *Route {
	return a.addRoute(httpMethodPatch, path, handler)
}

func (a *App) DELETE(path string, handler HandlerFunc) *Route {
	return a.addRoute(httpMethodDelete, path, handler)
}

func (a *App) HEAD(path string, handler HandlerFunc) *Route {
	return a.addRoute(httpMethodHead, path, handler)
}

func (a *App) OPTIONS(path string, handler HandlerFunc) *Route {
	return a.addRoute(httpMethodOptions, path, handler)
}

func (a *App) CONNECT(path string, handler HandlerFunc) *Route {
	return a.addRoute(httpMethodConnect, path, handler)
}

func (a *App) TRACE(path string, handler HandlerFunc) *Route {
	return a.addRoute(httpMethodTrace, path, handler)
}

func (a *App) ANY(path string, handler HandlerFunc) []*Route {
	methods := []string{
		httpMethodGet,
		httpMethodPost,
		httpMethodPut,
		httpMethodPatch,
		httpMethodDelete,
		httpMethodHead,
		httpMethodOptions,
	}
	out := make([]*Route, 0, len(methods))
	for _, method := range methods {
		out = append(out, a.addRoute(method, path, handler))
	}
	return out
}

func (g *Group) GET(path string, handler HandlerFunc) *Route {
	return g.addRoute(httpMethodGet, path, handler)
}

func (g *Group) POST(path string, handler HandlerFunc) *Route {
	return g.addRoute(httpMethodPost, path, handler)
}

func (g *Group) PUT(path string, handler HandlerFunc) *Route {
	return g.addRoute(httpMethodPut, path, handler)
}

func (g *Group) PATCH(path string, handler HandlerFunc) *Route {
	return g.addRoute(httpMethodPatch, path, handler)
}

func (g *Group) DELETE(path string, handler HandlerFunc) *Route {
	return g.addRoute(httpMethodDelete, path, handler)
}

func (g *Group) HEAD(path string, handler HandlerFunc) *Route {
	return g.addRoute(httpMethodHead, path, handler)
}

func (g *Group) OPTIONS(path string, handler HandlerFunc) *Route {
	return g.addRoute(httpMethodOptions, path, handler)
}

func (g *Group) CONNECT(path string, handler HandlerFunc) *Route {
	return g.addRoute(httpMethodConnect, path, handler)
}

func (g *Group) TRACE(path string, handler HandlerFunc) *Route {
	return g.addRoute(httpMethodTrace, path, handler)
}

func (g *Group) ANY(path string, handler HandlerFunc) []*Route {
	methods := []string{
		httpMethodGet,
		httpMethodPost,
		httpMethodPut,
		httpMethodPatch,
		httpMethodDelete,
		httpMethodHead,
		httpMethodOptions,
	}
	out := make([]*Route, 0, len(methods))
	for _, method := range methods {
		out = append(out, g.addRoute(method, path, handler))
	}
	return out
}

const (
	httpMethodGet     = "GET"
	httpMethodPost    = "POST"
	httpMethodPut     = "PUT"
	httpMethodPatch   = "PATCH"
	httpMethodDelete  = "DELETE"
	httpMethodHead    = "HEAD"
	httpMethodOptions = "OPTIONS"
	httpMethodConnect = "CONNECT"
	httpMethodTrace   = "TRACE"
)
