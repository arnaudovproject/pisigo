// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigo

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

var contextPool = sync.Pool{
	New: func() any {
		return &Context{
			store: make(map[string]any),
		}
	},
}

// Context is request-scoped and may be reused from a pool after the handler
// returns. Do not retain or access a Context from another goroutine after the
// handler returns unless you call DetachFromPool and later Release.
//
// Timeout middleware enables shared mode so the request goroutine and the
// handler goroutine may safely use the same Context until the handler finishes.
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
	holdDone        <-chan struct{}
	mu              sync.RWMutex
	shared          atomic.Bool
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
	c.shared.Store(false)
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
	c.holdDone = nil
	contextPool.Put(c)
}

// DetachFromPool prevents AdaptApp from returning this Context to the pool.
// The caller must invoke Release when the Context is no longer used.
func (c *Context) DetachFromPool() {
	c.skipPoolRelease = true
}

// HoldForHandler keeps the Context alive until done is signaled, after the
// middleware chain finishes. Used by Timeout so the request can return 504
// without waiting for a slow handler, while still protecting the pool.
func (c *Context) HoldForHandler(done <-chan struct{}) {
	c.holdDone = done
}

// EnableShared marks this Context safe for concurrent use by Timeout's
// handler goroutine and the request goroutine. Call DisableShared only after
// the handler goroutine has finished.
func (c *Context) EnableShared() {
	c.shared.Store(true)
}

// DisableShared ends concurrent Context access. Only call after the handler
// goroutine has returned.
func (c *Context) DisableShared() {
	c.shared.Store(false)
}

func (c *Context) lock() bool {
	if c.shared.Load() {
		c.mu.Lock()
		return true
	}
	return false
}

func (c *Context) unlock(locked bool) {
	if locked {
		c.mu.Unlock()
	}
}

func (c *Context) rlock() bool {
	if c.shared.Load() {
		c.mu.RLock()
		return true
	}
	return false
}

func (c *Context) runlock(locked bool) {
	if locked {
		c.mu.RUnlock()
	}
}

// Release returns a previously detached Context to the pool.
func (c *Context) Release() {
	releaseContext(c)
}

func (c *Context) reset(w http.ResponseWriter, r *http.Request, app *App, logger Logger, maxBody int64) {
	c.shared.Store(false)
	c.app = app
	c.writer = w
	c.request = r
	c.status = http.StatusOK
	c.written = false
	c.logger = logger
	c.body = nil
	c.bodyRead = false
	c.maxBody = maxBody
	c.skipPoolRelease = false
	c.holdDone = nil
	if c.store == nil {
		c.store = make(map[string]any)
	}
}

func (c *Context) Set(key string, value any) {
	locked := c.lock()
	defer c.unlock(locked)
	c.store[key] = value
}

func (c *Context) Get(key string) (any, bool) {
	locked := c.rlock()
	defer c.runlock(locked)
	value, ok := c.store[key]
	return value, ok
}

func (c *Context) MustGet(key string) any {
	locked := c.rlock()
	defer c.runlock(locked)
	value, ok := c.store[key]
	if !ok {
		panic("pisigo: key not found in context store: " + key)
	}
	return value
}

func (c *Context) Written() bool {
	locked := c.rlock()
	defer c.runlock(locked)
	// Prefer the recorder: Timeout may commit via timeoutWriter while c.written
	// is still false, and recorder access is mutex-protected.
	if rec := findRecorder(c.writer); rec != nil {
		_, written := rec.snapshot()
		if written {
			return true
		}
	}
	return c.written
}

func (c *Context) StatusCode() int {
	locked := c.rlock()
	defer c.runlock(locked)
	if rec := findRecorder(c.writer); rec != nil {
		status, written := rec.snapshot()
		if written {
			return status
		}
	}
	return c.status
}

// MarkWritten records that a response was already committed (status + body),
// so the error handler will not write again.
func (c *Context) MarkWritten(status int) {
	locked := c.lock()
	defer c.unlock(locked)
	if status != 0 {
		c.status = status
	}
	c.written = true
	if rec := findRecorder(c.writer); rec != nil {
		rec.mark(status)
	}
}

// syncFromWriter copies status/written flags from the response recorder after
// helpers that write through net/http directly (ServeFile, FileServer, …).
func (c *Context) syncFromWriter() {
	locked := c.lock()
	defer c.unlock(locked)
	if rec := findRecorder(c.writer); rec != nil {
		status, written := rec.snapshot()
		c.written = written
		if written {
			c.status = status
		}
		return
	}
	c.written = true
}

// IsHTTPS reports whether the request arrived over TLS, or was forwarded as
// HTTPS by a trusted proxy (X-Forwarded-Proto). Untrusted clients cannot
// spoof HTTPS via X-Forwarded-Proto alone.
func (c *Context) IsHTTPS() bool {
	if c.request != nil && c.request.TLS != nil {
		return true
	}
	if c.app == nil || len(c.app.trustedProxies) == 0 || c.request == nil {
		return false
	}
	remote := remoteIP(c.request.RemoteAddr)
	if !ipTrusted(remote, c.app.trustedProxies) {
		return false
	}
	proto := c.request.Header.Get("X-Forwarded-Proto")
	if i := strings.IndexByte(proto, ','); i >= 0 {
		proto = proto[:i]
	}
	return strings.EqualFold(strings.TrimSpace(proto), "https")
}

func (c *Context) Log() Logger {
	locked := c.rlock()
	defer c.runlock(locked)
	if c.logger == nil {
		return DefaultLogger()
	}
	return c.logger
}

func (c *Context) SetLogger(logger Logger) {
	if logger == nil {
		return
	}
	locked := c.lock()
	defer c.unlock(locked)
	c.logger = logger
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
