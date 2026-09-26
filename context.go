// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigo

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"sync"
)

var contextPool = sync.Pool{
	New: func() any {
		return &Context{
			store: make(map[string]any),
		}
	},
}

type Context struct {
	app             *App
	writer          http.ResponseWriter
	request         *http.Request
	status          int
	written         bool
	store           map[string]any
	logger          Logger
	body            []byte
	bodyRead        bool
	maxBody         int64
	skipPoolRelease bool
}

func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	c := contextPool.Get().(*Context)
	c.reset(w, r, nil, DefaultLogger(), 0)
	return c
}

func acquireContext(app *App, w http.ResponseWriter, r *http.Request, logger Logger, maxBody int64) *Context {
	c := contextPool.Get().(*Context)
	c.reset(w, r, app, logger, maxBody)
	return c
}

func releaseContext(c *Context) {
	for k := range c.store {
		delete(c.store, k)
	}
	c.app = nil
	c.writer = nil
	c.request = nil
	c.logger = nil
	c.body = nil
	c.bodyRead = false
	c.status = http.StatusOK
	c.written = false
	c.maxBody = 0
	c.skipPoolRelease = false
	contextPool.Put(c)
}

// DetachFromPool prevents AdaptApp from returning this Context to the pool.
// The caller must invoke Release when the Context is no longer used.
func (c *Context) DetachFromPool() {
	c.skipPoolRelease = true
}

// Release returns a previously detached Context to the pool.
func (c *Context) Release() {
	releaseContext(c)
}

func (c *Context) reset(w http.ResponseWriter, r *http.Request, app *App, logger Logger, maxBody int64) {
	c.app = app
	c.writer = w
	c.request = r
	c.status = http.StatusOK
	c.written = false
	c.logger = logger
	c.body = nil
	c.bodyRead = false
	c.maxBody = maxBody
	if c.store == nil {
		c.store = make(map[string]any)
	}
}

func (c *Context) Set(key string, value any) {
	c.store[key] = value
}

func (c *Context) Get(key string) (any, bool) {
	value, ok := c.store[key]
	return value, ok
}

func (c *Context) MustGet(key string) any {
	value, ok := c.store[key]
	if !ok {
		panic("pisigo: key not found in context store: " + key)
	}
	return value
}

func (c *Context) Written() bool {
	return c.written
}

func (c *Context) StatusCode() int {
	return c.status
}

func (c *Context) Log() Logger {
	if c.logger == nil {
		return DefaultLogger()
	}
	return c.logger
}

func (c *Context) SetLogger(logger Logger) {
	if logger != nil {
		c.logger = logger
	}
}

func (c *Context) App() *App {
	return c.app
}

func (c *Context) readBody() ([]byte, error) {
	if c.bodyRead {
		return c.body, nil
	}
	c.bodyRead = true
	var reader io.Reader = c.request.Body
	if c.maxBody > 0 {
		reader = io.LimitReader(c.request.Body, c.maxBody+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if c.maxBody > 0 && int64(len(data)) > c.maxBody {
		return nil, ErrRequestEntityTooLarge
	}
	c.body = data
	c.request.Body = io.NopCloser(bytes.NewReader(data))
	return data, nil
}

func (c *Context) IP() string {
	if c.app != nil {
		return c.clientIP(c.app.trustedProxies, c.app.proxyHeaders)
	}
	host, _, err := net.SplitHostPort(c.request.RemoteAddr)
	if err != nil {
		return c.request.RemoteAddr
	}
	return host
}
