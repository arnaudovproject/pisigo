package session

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/arnaudovproject/pisigo"
)

const CookieName = "pisigo_session"

type Store interface {
	Get(id string) (map[string]any, bool)
	Save(id string, data map[string]any, ttl time.Duration) error
	Delete(id string) error
}

type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]memoryItem
}

type memoryItem struct {
	values map[string]any
	expire time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]memoryItem)}
}

func (s *MemoryStore) Get(id string) (map[string]any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.data[id]
	if !ok || (!item.expire.IsZero() && time.Now().After(item.expire)) {
		return nil, false
	}
	out := make(map[string]any, len(item.values))
	for k, v := range item.values {
		out[k] = v
	}
	return out, true
}

func (s *MemoryStore) Save(id string, data map[string]any, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make(map[string]any, len(data))
	for k, v := range data {
		copied[k] = v
	}
	item := memoryItem{values: copied}
	if ttl > 0 {
		item.expire = time.Now().Add(ttl)
	}
	s.data[id] = item
	return nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

type Config struct {
	Store Store
	Cookie string
	TTL    time.Duration
	// Secure marks the session cookie as Secure. Ignored when InsecureCookie is true.
	// When both are false (zero Config), Secure defaults to true.
	Secure bool
	// InsecureCookie forces Secure=false for local HTTP development.
	InsecureCookie bool
	HTTPOnly       bool
	Path           string
	SameSite       http.SameSite
}

func Middleware(cfg Config) pisigo.Middleware {
	if cfg.Store == nil {
		cfg.Store = NewMemoryStore()
	}
	if cfg.Cookie == "" {
		cfg.Cookie = CookieName
	}
	if cfg.TTL == 0 {
		cfg.TTL = 24 * time.Hour
	}
	if cfg.Path == "" {
		cfg.Path = "/"
	}
	if cfg.SameSite == 0 {
		cfg.SameSite = http.SameSiteLaxMode
	}
	cfg.HTTPOnly = true
	if cfg.InsecureCookie {
		cfg.Secure = false
	} else {
		cfg.Secure = true
	}

	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			id := ""
			if cookie, err := c.CookieGet(cfg.Cookie); err == nil {
				id = cookie.Value
			}
			values, ok := map[string]any{}, false
			if id != "" {
				values, ok = cfg.Store.Get(id)
			}
			if !ok {
				id = newID()
				values = map[string]any{}
			}
			sess := &Session{
				id:         id,
				values:     values,
				store:      cfg.Store,
				ttl:        cfg.TTL,
				dirty:      !ok,
				cookieName: cfg.Cookie,
				cookiePath: cfg.Path,
			}
			c.Set("session", sess)
			writeCookie := func() {
				c.Cookie(&http.Cookie{
					Name:     cfg.Cookie,
					Value:    sess.id,
					Path:     cfg.Path,
					MaxAge:   int(cfg.TTL.Seconds()),
					HttpOnly: cfg.HTTPOnly,
					Secure:   cfg.Secure,
					SameSite: cfg.SameSite,
				})
			}
			if sess.dirty {
				writeCookie()
			}
			err := next(c)
			if sess.dirty {
				_ = cfg.Store.Save(sess.id, sess.values, cfg.TTL)
				writeCookie()
			}
			return err
		}
	}
}

type Session struct {
	id         string
	values     map[string]any
	store      Store
	ttl        time.Duration
	dirty      bool
	cookieName string
	cookiePath string
}

func FromContext(c *pisigo.Context) (*Session, bool) {
	raw, ok := c.Get("session")
	if !ok {
		return nil, false
	}
	s, ok := raw.(*Session)
	return s, ok
}

func (s *Session) Get(key string) (any, bool) {
	v, ok := s.values[key]
	return v, ok
}

func (s *Session) Set(key string, value any) {
	s.values[key] = value
	s.dirty = true
}

func (s *Session) Delete(key string) {
	delete(s.values, key)
	s.dirty = true
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) Destroy(c *pisigo.Context) {
	_ = s.store.Delete(s.id)
	s.values = map[string]any{}
	s.dirty = false
	name := s.cookieName
	if name == "" {
		name = CookieName
	}
	path := s.cookiePath
	if path == "" {
		path = "/"
	}
	c.DeleteCookie(name, path)
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
