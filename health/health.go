package health

import (
	"context"
	"sync"
	"time"

	"github.com/arnaudovproject/pisigo"
)

type CheckFunc func(ctx context.Context) error

type Checker struct {
	mu     sync.RWMutex
	live   map[string]CheckFunc
	ready  map[string]CheckFunc
}

func New() *Checker {
	return &Checker{
		live:  map[string]CheckFunc{},
		ready: map[string]CheckFunc{},
	}
}

func (c *Checker) Live(name string, fn CheckFunc) {
	c.mu.Lock()
	c.live[name] = fn
	c.mu.Unlock()
}

func (c *Checker) Ready(name string, fn CheckFunc) {
	c.mu.Lock()
	c.ready[name] = fn
	c.mu.Unlock()
}

func (c *Checker) Register(app *pisigo.App, prefix string) {
	if prefix == "" {
		prefix = "/health"
	}
	app.GET(prefix+"/live", c.LiveHandler())
	app.GET(prefix+"/ready", c.ReadyHandler())
	app.GET(prefix, c.ReadyHandler())
}

func (c *Checker) LiveHandler() pisigo.HandlerFunc {
	return c.handle(func() map[string]CheckFunc {
		c.mu.RLock()
		defer c.mu.RUnlock()
		out := map[string]CheckFunc{}
		for k, v := range c.live {
			out[k] = v
		}
		return out
	})
}

func (c *Checker) ReadyHandler() pisigo.HandlerFunc {
	return c.handle(func() map[string]CheckFunc {
		c.mu.RLock()
		defer c.mu.RUnlock()
		out := map[string]CheckFunc{}
		for k, v := range c.ready {
			out[k] = v
		}
		for k, v := range c.live {
			out[k] = v
		}
		return out
	})
}

func (c *Checker) handle(getChecks func() map[string]CheckFunc) pisigo.HandlerFunc {
	return func(ctx *pisigo.Context) error {
		checks := getChecks()
		results := map[string]string{}
		ok := true
		reqCtx, cancel := context.WithTimeout(ctx.Request().Context(), 3*time.Second)
		defer cancel()
		for name, fn := range checks {
			if fn == nil {
				results[name] = "ok"
				continue
			}
			if err := fn(reqCtx); err != nil {
				results[name] = err.Error()
				ok = false
			} else {
				results[name] = "ok"
			}
		}
		status := 200
		state := "up"
		if !ok {
			status = 503
			state = "down"
		}
		return ctx.JSON(status, map[string]any{"status": state, "checks": results})
	}
}
