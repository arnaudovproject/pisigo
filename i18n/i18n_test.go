// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package i18n_test

import (
	"testing"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/i18n"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestBundleTranslateAndFallback(t *testing.T) {
	b := i18n.New("en")
	b.Add("en", map[string]string{"hello": "Hello {name}"})
	b.Add("bg", map[string]string{"hello": "Здравей {name}"})
	if got := b.T("bg", "hello", "name", "V"); got != "Здравей V" {
		t.Fatalf("got %q", got)
	}
	if got := b.T("en-US", "hello", "name", "V"); got != "Hello V" {
		t.Fatalf("locale fallback got %q", got)
	}
	if got := b.T("fr", "hello", "name", "V"); got != "Hello V" {
		t.Fatalf("fallback got %q", got)
	}
	if got := b.T("en", "missing"); got != "missing" {
		t.Fatalf("key=%q", got)
	}
}

func TestI18nMiddleware(t *testing.T) {
	b := i18n.New("en")
	b.Add("en", map[string]string{"hi": "Hi"})
	app := pisigo.Boot()
	app.Use(b.Middleware(""))
	app.GET("/", func(c *pisigo.Context) error {
		return c.String(200, i18n.T(c, "hi"))
	})
	res := pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/",
		Headers: map[string]string{"Accept-Language": "en-US,en;q=0.9"},
	})
	if res.String() != "Hi" {
		t.Fatalf("got %q", res.String())
	}
}
