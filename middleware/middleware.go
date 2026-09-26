// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/arnaudovproject/pisigo"
)

func Recover() pisigo.Middleware {
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					c.Log().Error("panic recovered",
						"panic", fmt.Sprint(rec),
						"stack", string(debug.Stack()),
						"method", c.Method(),
						"path", c.Path(),
					)
					err = pisigo.ErrInternalServer
				}
			}()
			return next(c)
		}
	}
}

func Logger() pisigo.Middleware {
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			start := time.Now()
			err := next(c)
			args := []any{
				"method", c.Method(),
				"path", c.Path(),
				"status", c.StatusCode(),
				"latency", time.Since(start).String(),
				"ip", c.IP(),
			}
			if err != nil {
				args = append(args, "error", err)
				c.Log().Error("request", args...)
				return err
			}
			status := c.StatusCode()
			switch {
			case status >= 500:
				c.Log().Error("request", args...)
			case status >= 400:
				c.Log().Warn("request", args...)
			default:
				c.Log().Info("request", args...)
			}
			return nil
		}
	}
}

const RequestIDKey = "pisigo_request_id"
const RequestIDHeader = "X-Request-ID"

func RequestID() pisigo.Middleware {
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			id := c.HeaderGet(RequestIDHeader)
			if id == "" {
				id = newRequestID()
			}
			c.Set(RequestIDKey, id)
			c.Header(RequestIDHeader, id)
			c.SetLogger(c.Log().With("request_id", id))
			return next(c)
		}
	}
}

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", RequestIDHeader},
		MaxAge:       86400,
	}
}

func CORS(cfg ...CORSConfig) pisigo.Middleware {
	config := DefaultCORSConfig()
	if len(cfg) > 0 {
		config = cfg[0]
	}
	allowOrigin := "*"
	if len(config.AllowOrigins) == 1 {
		allowOrigin = config.AllowOrigins[0]
	}
	allowMethods := join(config.AllowMethods)
	allowHeaders := join(config.AllowHeaders)
	exposeHeaders := join(config.ExposeHeaders)

	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			origin := c.HeaderGet("Origin")
			acaOrigin := ""
			wildcard := allowOrigin == "*" || contains(config.AllowOrigins, "*")
			// Credentials + wildcard is unsafe (reflects any Origin). Require an explicit allowlist.
			if config.AllowCredentials && wildcard {
				wildcard = false
			}
			if origin != "" {
				if wildcard {
					acaOrigin = "*"
				} else if contains(config.AllowOrigins, origin) {
					acaOrigin = origin
				} else if allowOrigin != "*" && allowOrigin != "" && origin == allowOrigin {
					acaOrigin = origin
				}
			} else if allowOrigin != "" && !config.AllowCredentials {
				acaOrigin = allowOrigin
			}
			if acaOrigin != "" {
				c.Header("Access-Control-Allow-Origin", acaOrigin)
				if acaOrigin != "*" {
					c.Header("Vary", "Origin")
				}
			}

			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			if exposeHeaders != "" {
				c.Header("Access-Control-Expose-Headers", exposeHeaders)
			}

			if c.Method() == http.MethodOptions {
				c.Header("Access-Control-Allow-Methods", allowMethods)
				c.Header("Access-Control-Allow-Headers", allowHeaders)
				if config.MaxAge > 0 {
					c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
				}
				return c.NoContent()
			}
			return next(c)
		}
	}
}

func Timeout(d time.Duration) pisigo.Middleware {
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			ctx, cancel := context.WithTimeout(c.Request().Context(), d)
			defer cancel()
			c.SetRequest(c.Request().WithContext(ctx))

			orig := c.Response()
			tw := &timeoutWriter{ResponseWriter: orig}
			c.SetWriter(tw)

			done := make(chan error, 1)
			go func() {
				defer func() {
					if rec := recover(); rec != nil {
						done <- pisigo.ErrInternalServer
					}
				}()
				done <- next(c)
			}()

			select {
			case err := <-done:
				c.SetWriter(orig)
				return err
			case <-ctx.Done():
				tw.mu.Lock()
				tw.timedOut = true
				tw.mu.Unlock()
				c.SetWriter(orig)
				// Wait for the handler goroutine so the pooled Context is not
				// released while next(c) is still running.
				<-done
				return pisigo.ErrGatewayTimeout
			}
		}
	}
}

type timeoutWriter struct {
	http.ResponseWriter
	mu       sync.Mutex
	timedOut bool
}

func (w *timeoutWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *timeoutWriter) WriteHeader(status int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *timeoutWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}

func Secure() pisigo.Middleware {
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			c.Header("X-Content-Type-Options", "nosniff")
			c.Header("X-Frame-Options", "DENY")
			c.Header("X-XSS-Protection", "0")
			c.Header("Referrer-Policy", "no-referrer")
			c.Header("Content-Security-Policy", "default-src 'self'")
			c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			if c.Request().TLS != nil || strings.EqualFold(c.HeaderGet("X-Forwarded-Proto"), "https") {
				c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			return next(c)
		}
	}
}

type BasicAuthConfig struct {
	Validator func(username, password string) bool
	Realm     string
}

func BasicAuth(cfg BasicAuthConfig) pisigo.Middleware {
	realm := cfg.Realm
	if realm == "" {
		realm = "Restricted"
	}
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			user, pass, ok := c.Request().BasicAuth()
			if !ok || cfg.Validator == nil || !cfg.Validator(user, pass) {
				c.Header("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
				return pisigo.ErrUnauthorized
			}
			c.Set("basic_auth_user", user)
			return next(c)
		}
	}
}

type RateLimitConfig struct {
	Limit   int
	Window  time.Duration
	KeyFunc func(c *pisigo.Context) string
	Store   RateLimitStore
}

type RateLimitStore interface {
	Allow(key string, limit int, window time.Duration) (bool, error)
}

func RateLimit(cfg RateLimitConfig) pisigo.Middleware {
	if cfg.Limit <= 0 {
		cfg.Limit = 100
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *pisigo.Context) string { return c.IP() }
	}
	if cfg.Store == nil {
		cfg.Store = newMemoryRateLimitStore()
	}
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			ok, err := cfg.Store.Allow(cfg.KeyFunc(c), cfg.Limit, cfg.Window)
			if err != nil {
				c.Log().Error("rate limit store error", "error", err)
				return pisigo.ErrServiceUnavailable
			}
			if !ok {
				return pisigo.ErrTooManyRequests
			}
			return next(c)
		}
	}
}

func join(values []string) string {
	if len(values) == 0 {
		return ""
	}
	out := values[0]
	for i := 1; i < len(values); i++ {
		out += ", " + values[i]
	}
	return out
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
