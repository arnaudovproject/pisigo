package pisigo

import (
	"net"
	"net/http"
	"path"
	"strings"
)

type App struct {
	routes           []Route
	middlewares      []Middleware
	logger           Logger
	errorHandler     ErrorHandler
	notFound         HandlerFunc
	methodNotAllowed HandlerFunc
	trustedProxies   []*net.IPNet
	proxyHeaders     []string
	maxBodyBytes     int64
	namedRoutes      map[string]*Route
	httpServer       *http.Server
}

func Boot() *App {
	return &App{
		logger:       DefaultLogger(),
		errorHandler: DefaultErrorHandler,
		maxBodyBytes: 1 << 20,
		namedRoutes:  make(map[string]*Route),
		proxyHeaders: DefaultProxyConfig().Headers,
		notFound: func(c *Context) error {
			return ErrNotFound
		},
		methodNotAllowed: func(c *Context) error {
			return ErrMethodNotAllowed
		},
	}
}

func (a *App) Use(middlewares ...Middleware) {
	a.middlewares = append(a.middlewares, middlewares...)
}

func (a *App) SetLogger(logger Logger) {
	if logger != nil {
		a.logger = logger
	}
}

func (a *App) Logger() Logger {
	return a.logger
}

func (a *App) SetErrorHandler(handler ErrorHandler) {
	if handler != nil {
		a.errorHandler = handler
	}
}

func (a *App) SetNotFound(handler HandlerFunc) {
	if handler != nil {
		a.notFound = handler
	}
}

func (a *App) SetMethodNotAllowed(handler HandlerFunc) {
	if handler != nil {
		a.methodNotAllowed = handler
	}
}

func (a *App) SetMaxBodyBytes(n int64) {
	if n > 0 {
		a.maxBodyBytes = n
	}
}

func (a *App) Routes() []Route {
	out := make([]Route, len(a.routes))
	copy(out, a.routes)
	return out
}

func (a *App) Reverse(name string, params ...string) (string, error) {
	route, ok := a.namedRoutes[name]
	if !ok {
		return "", NewHTTPError(http.StatusNotFound, "route not found: "+name)
	}
	p := route.Path
	for i := 0; i+1 < len(params); i += 2 {
		p = strings.ReplaceAll(p, "{"+params[i]+"}", params[i+1])
	}
	if strings.Contains(p, "{") {
		return "", NewHTTPError(http.StatusBadRequest, "missing route params for: "+name)
	}
	return p, nil
}

func (a *App) Static(prefix, root string) {
	prefix = "/" + strings.Trim(prefix, "/")
	root = path.Clean(root)
	fileServer := http.StripPrefix(prefix, http.FileServer(http.Dir(root)))
	a.GET(prefix+"/{path...}", func(c *Context) error {
		fileServer.ServeHTTP(c.writer, c.request)
		c.syncFromWriter()
		return nil
	})
}

func (a *App) File(urlPath, filePath string) {
	a.GET(urlPath, func(c *Context) error {
		return c.File(filePath)
	})
}

func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()

	for i := range a.routes {
		route := a.routes[i]
		handler := route.Handler
		for i := len(a.middlewares) - 1; i >= 0; i-- {
			handler = a.middlewares[i](handler)
		}
		pattern := route.Method + " " + route.Path
		mux.Handle(pattern, a.adapt(handler))
	}

	wrap := func(handler HandlerFunc) HandlerFunc {
		for i := len(a.middlewares) - 1; i >= 0; i-- {
			handler = a.middlewares[i](handler)
		}
		return handler
	}
	notFound := a.adapt(wrap(a.notFound))
	methodNotAllowed := a.adapt(wrap(a.methodNotAllowed))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)
		if pattern == "" {
			// Probe other methods via ServeMux's own matcher (O(methods),
			// not O(routes)) so unmatched bot traffic stays cheap.
			if otherMethodMatches(mux, r) {
				methodNotAllowed.ServeHTTP(w, r)
				return
			}
			notFound.ServeHTTP(w, r)
			return
		}
		// ServeMux.ServeHTTP populates Request.PathValue; mux.Handler does not.
		mux.ServeHTTP(w, r)
	})
}

var probeHTTPMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
	http.MethodConnect,
	http.MethodTrace,
}

func otherMethodMatches(mux *http.ServeMux, r *http.Request) bool {
	for _, method := range probeHTTPMethods {
		if method == r.Method {
			continue
		}
		probe := r.Clone(r.Context())
		probe.Method = method
		if _, pattern := mux.Handler(probe); pattern != "" {
			return true
		}
	}
	return false
}

func (a *App) adapt(handler HandlerFunc) http.Handler {
	return AdaptApp(a, handler)
}
