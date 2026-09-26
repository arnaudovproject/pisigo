// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package middleware

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/store"
)

type CacheConfig struct {
	Store   store.Store
	TTL     time.Duration
	KeyFunc func(c *pisigo.Context) string
	Methods []string
}

type cacheEntry struct {
	Status      int               `json:"status"`
	ContentType string            `json:"content_type"`
	Body        string            `json:"body"`
	Headers     map[string]string `json:"headers,omitempty"`
}

func Cache(cfg CacheConfig) pisigo.Middleware {
	if cfg.TTL <= 0 {
		cfg.TTL = time.Minute
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *pisigo.Context) string {
			sum := sha1.Sum([]byte(c.Method() + ":" + c.Path() + "?" + c.Request().URL.RawQuery))
			return "pisigo:cache:" + hex.EncodeToString(sum[:])
		}
	}
	allowed := map[string]bool{}
	if len(cfg.Methods) == 0 {
		allowed["GET"] = true
		allowed["HEAD"] = true
	} else {
		for _, m := range cfg.Methods {
			allowed[m] = true
		}
	}

	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			if cfg.Store == nil || !allowed[c.Method()] {
				return next(c)
			}
			key := cfg.KeyFunc(c)
			if val, err := cfg.Store.Get(c.Request().Context(), key); err == nil {
				var entry cacheEntry
				if json.Unmarshal([]byte(val), &entry) != nil {
					entry = cacheEntry{Status: 200, ContentType: "application/octet-stream", Body: val}
				}
				c.Header("X-Cache", "HIT")
				for k, v := range entry.Headers {
					c.Header(k, v)
				}
				return c.Data(entry.Status, entry.ContentType, []byte(entry.Body))
			}

			orig := c.Response()
			rec := &bodyRecorder{ResponseWriter: orig, status: 200}
			c.SetWriter(rec)
			c.Header("X-Cache", "MISS")
			err := next(c)
			c.SetWriter(orig)
			if err == nil && rec.status >= 200 && rec.status < 300 && len(rec.body) > 0 {
				ct := rec.Header().Get("Content-Type")
				if ct == "" {
					ct = "application/octet-stream"
				}
				payload, _ := json.Marshal(cacheEntry{
					Status:      rec.status,
					ContentType: ct,
					Body:        string(rec.body),
				})
				_ = cfg.Store.Set(c.Request().Context(), key, string(payload), cfg.TTL)
			}
			return err
		}
	}
}

type bodyRecorder struct {
	http.ResponseWriter
	status int
	body   []byte
}

func (b *bodyRecorder) WriteHeader(status int) {
	b.status = status
	b.ResponseWriter.WriteHeader(status)
}

func (b *bodyRecorder) Write(p []byte) (int, error) {
	b.body = append(b.body, p...)
	return b.ResponseWriter.Write(p)
}
