package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

	"github.com/arnaudovproject/pisigo"
)

const CSRFTokenKey = "csrf_token"
const CSRFHeader = "X-CSRF-Token"
const CSRFFormField = "csrf_token"

type CSRFConfig struct {
	CookieName string
	HeaderName string
	FormField  string
	// Secure marks the CSRF cookie as Secure. Defaults to true when using CSRF().
	// Pass CSRF(CSRFConfig{Secure: false}) for plain HTTP local development.
	Secure bool
}

func CSRF(cfg ...CSRFConfig) pisigo.Middleware {
	config := CSRFConfig{
		CookieName: "pisigo_csrf",
		HeaderName: CSRFHeader,
		FormField:  CSRFFormField,
		Secure:     true,
	}
	if len(cfg) > 0 {
		if cfg[0].CookieName != "" {
			config.CookieName = cfg[0].CookieName
		}
		if cfg[0].HeaderName != "" {
			config.HeaderName = cfg[0].HeaderName
		}
		if cfg[0].FormField != "" {
			config.FormField = cfg[0].FormField
		}
		config.Secure = cfg[0].Secure
	}

	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			token := ""
			if cookie, err := c.CookieGet(config.CookieName); err == nil {
				token = cookie.Value
			}
			if token == "" {
				token = newToken()
				c.Cookie(&http.Cookie{
					Name:     config.CookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false,
					Secure:   config.Secure,
					SameSite: http.SameSiteLaxMode,
				})
			}
			c.Set(CSRFTokenKey, token)
			c.Header("X-CSRF-Token", token)

			switch c.Method() {
			case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
				return next(c)
			}

			provided := c.HeaderGet(config.HeaderName)
			if provided == "" {
				provided = c.FormValue(config.FormField)
			}
			if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
				return pisigo.ErrForbidden
			}
			return next(c)
		}
	}
}

func newToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
