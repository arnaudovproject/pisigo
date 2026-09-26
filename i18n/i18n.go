// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package i18n

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/arnaudovproject/pisigo"
)

type Bundle struct {
	mu       sync.RWMutex
	messages map[string]map[string]string
	fallback string
}

func New(fallback string) *Bundle {
	if fallback == "" {
		fallback = "en"
	}
	return &Bundle{
		messages: map[string]map[string]string{},
		fallback: fallback,
	}
}

func (b *Bundle) Add(locale string, messages map[string]string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.messages[locale] == nil {
		b.messages[locale] = map[string]string{}
	}
	for k, v := range messages {
		b.messages[locale][k] = v
	}
}

func (b *Bundle) LoadJSON(locale, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	messages := map[string]string{}
	if err := json.Unmarshal(data, &messages); err != nil {
		return err
	}
	b.Add(locale, messages)
	return nil
}

func (b *Bundle) T(locale, key string, vars ...string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	msg, ok := b.lookup(locale, key)
	if !ok {
		if i := strings.IndexByte(locale, '-'); i > 0 {
			msg, ok = b.lookup(locale[:i], key)
		}
	}
	if !ok {
		msg, ok = b.lookup(b.fallback, key)
	}
	if !ok {
		return key
	}
	for i := 0; i+1 < len(vars); i += 2 {
		msg = strings.ReplaceAll(msg, "{"+vars[i]+"}", vars[i+1])
	}
	return msg
}

func (b *Bundle) lookup(locale, key string) (string, bool) {
	msgs, ok := b.messages[locale]
	if !ok {
		return "", false
	}
	msg, ok := msgs[key]
	return msg, ok
}

func (b *Bundle) Middleware(defaultLocale string) pisigo.Middleware {
	if defaultLocale == "" {
		defaultLocale = b.fallback
	}
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			locale := c.QueryDefault("lang", "")
			if locale == "" {
				locale = c.HeaderGet("Accept-Language")
				if i := strings.IndexByte(locale, ','); i >= 0 {
					locale = locale[:i]
				}
				if i := strings.IndexByte(locale, ';'); i >= 0 {
					locale = locale[:i]
				}
				locale = strings.TrimSpace(locale)
			}
			if locale == "" {
				locale = defaultLocale
			}
			c.Set("locale", locale)
			c.Set("i18n", b)
			return next(c)
		}
	}
}

func T(c *pisigo.Context, key string, vars ...string) string {
	locale, _ := c.Get("locale")
	loc, _ := locale.(string)
	raw, ok := c.Get("i18n")
	if !ok {
		return key
	}
	b, _ := raw.(*Bundle)
	if b == nil {
		return key
	}
	return b.T(loc, key, vars...)
}
