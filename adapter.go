// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigo

import (
	"net/http"
)

type HandlerFunc func(*Context) error

type Middleware func(HandlerFunc) HandlerFunc

func Adapter(handler HandlerFunc) http.Handler {
	return Adapt(handler, DefaultLogger())
}

func Adapt(handler HandlerFunc, logger Logger) http.Handler {
	app := Boot()
	app.SetLogger(logger)
	return AdaptApp(app, handler)
}

func AdaptApp(app *App, handler HandlerFunc) http.Handler {
	logger := app.logger
	if logger == nil {
		logger = DefaultLogger()
	}
	errorHandler := app.errorHandler
	if errorHandler == nil {
		errorHandler = DefaultErrorHandler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := acquireContext(app, w, r, logger, app.maxBodyBytes)
		defer func() {
			if !ctx.skipPoolRelease {
				releaseContext(ctx)
			}
		}()
		if err := handler(ctx); err != nil {
			errorHandler(ctx, err)
		}
	})
}

func Chain(handler HandlerFunc, middlewares ...Middleware) HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
