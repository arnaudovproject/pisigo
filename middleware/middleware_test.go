// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package middleware_test

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/adapters/memory"
	"github.com/arnaudovproject/pisigo/middleware"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestRecover(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.Recover())
	app.GET("/panic", func(c *pisigo.Context) error {
		panic("boom")
	})
	res := pisigotest.GET(app, "/panic")
	if res.Code != 500 {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestRequestID(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.RequestID())
	app.GET("/", func(c *pisigo.Context) error { return c.NoContent() })
	res := pisigotest.GET(app, "/")
	if res.Header.Get(middleware.RequestIDHeader) == "" {
		t.Fatal("missing request id")
	}
	res = pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/",
		Headers: map[string]string{middleware.RequestIDHeader: "custom-id"},
	})
	if res.Header.Get(middleware.RequestIDHeader) != "custom-id" {
		t.Fatalf("got %q", res.Header.Get(middleware.RequestIDHeader))
	}
}

func TestRequestIDRejectsInvalid(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.RequestID())
	app.GET("/", func(c *pisigo.Context) error { return c.NoContent() })

	res := pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/",
		Headers: map[string]string{middleware.RequestIDHeader: "bad id with spaces!"},
	})
	got := res.Header.Get(middleware.RequestIDHeader)
	if got == "" || got == "bad id with spaces!" {
		t.Fatalf("expected generated id, got %q", got)
	}

	huge := strings.Repeat("a", 200)
	res = pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/",
		Headers: map[string]string{middleware.RequestIDHeader: huge},
	})
	got = res.Header.Get(middleware.RequestIDHeader)
	if got == huge || len(got) > 128 {
		t.Fatalf("expected capped generated id, got len=%d", len(got))
	}
}

func TestCORS(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins:     []string{"https://app.example"},
		AllowCredentials: true,
	}))
	app.GET("/", func(c *pisigo.Context) error { return c.String(200, "ok") })

	res := pisigotest.Do(app, pisigotest.Request{
		Method:  "OPTIONS",
		Path:    "/",
		Headers: map[string]string{"Origin": "https://app.example"},
	})
	if res.Code != 204 {
		t.Fatalf("preflight=%d", res.Code)
	}
	if res.Header.Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Fatalf("origin=%q", res.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSCredentialsWildcard(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	}))
	app.GET("/", func(c *pisigo.Context) error { return c.NoContent() })
	res := pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/",
		Headers: map[string]string{"Origin": "https://x.test"},
	})
	// Credentials + wildcard must not reflect arbitrary origins.
	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no ACAO with credentials+*, got %q", got)
	}
}

func TestTimeout(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.Timeout(50 * time.Millisecond))
	app.GET("/slow", func(c *pisigo.Context) error {
		select {
		case <-c.Request().Context().Done():
			return c.Request().Context().Err()
		case <-time.After(500 * time.Millisecond):
			return c.String(200, "late")
		}
	})
	res := pisigotest.GET(app, "/slow")
	if res.Code != 504 {
		t.Fatalf("status=%d body=%s", res.Code, res.String())
	}
}

func TestTimeoutDoesNotBlockOnSlowHandler(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.Timeout(40 * time.Millisecond))
	finished := make(chan struct{})
	app.GET("/ignore-cancel", func(c *pisigo.Context) error {
		time.Sleep(200 * time.Millisecond)
		close(finished)
		return c.String(200, "late")
	})
	start := time.Now()
	res := pisigotest.GET(app, "/ignore-cancel")
	elapsed := time.Since(start)
	if res.Code != 504 {
		t.Fatalf("status=%d body=%s", res.Code, res.String())
	}
	if elapsed > 120*time.Millisecond {
		t.Fatalf("timeout blocked waiting for handler: %v", elapsed)
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("handler never finished")
	}
}

func TestSecureBasicAuth(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.Secure(), middleware.BasicAuth(middleware.BasicAuthConfig{
		Validator: func(u, p string) bool { return u == "a" && p == "b" },
	}))
	app.GET("/", func(c *pisigo.Context) error { return c.String(200, "ok") })

	res := pisigotest.GET(app, "/")
	if res.Code != 401 {
		t.Fatalf("unauth=%d", res.Code)
	}
	res = pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/",
		Headers: map[string]string{"Authorization": "Basic YTpi"}, // a:b
	})
	if res.Code != 200 {
		t.Fatalf("auth=%d", res.Code)
	}
	if res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing secure header")
	}
}

func TestSecureHSTSRequiresTrustedProxy(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.Secure())
	app.GET("/", func(c *pisigo.Context) error { return c.NoContent() })

	res := pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/",
		Headers: map[string]string{"X-Forwarded-Proto": "https"},
	})
	if res.Header.Get("Strict-Transport-Security") != "" {
		t.Fatal("untrusted client must not set HSTS via X-Forwarded-Proto")
	}

	app = pisigo.Boot()
	if err := app.SetTrustedProxies("192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	app.Use(middleware.Secure())
	app.GET("/", func(c *pisigo.Context) error { return c.NoContent() })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("expected HSTS from trusted proxy")
	}
}

func TestRateLimit(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.RateLimit(middleware.RateLimitConfig{Limit: 2, Window: time.Minute}))
	app.GET("/", func(c *pisigo.Context) error { return c.NoContent() })
	if pisigotest.GET(app, "/").Code != 204 {
		t.Fatal("1")
	}
	if pisigotest.GET(app, "/").Code != 204 {
		t.Fatal("2")
	}
	if pisigotest.GET(app, "/").Code != 429 {
		t.Fatal("expected 429")
	}
}

func TestCompress(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.Compress())
	app.GET("/", func(c *pisigo.Context) error {
		return c.String(200, strings.Repeat("x", 100))
	})
	res := pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/",
		Headers: map[string]string{"Accept-Encoding": "gzip"},
	})
	if res.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("encoding=%q", res.Header.Get("Content-Encoding"))
	}
	zr, err := gzip.NewReader(strings.NewReader(res.String()))
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	out, _ := io.ReadAll(zr)
	if len(out) != 100 {
		t.Fatalf("len=%d", len(out))
	}
}

func TestCSRF(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.CSRF())
	app.GET("/form", func(c *pisigo.Context) error { return c.NoContent() })
	app.POST("/form", func(c *pisigo.Context) error { return c.String(200, "ok") })

	get := pisigotest.GET(app, "/form")
	token := get.Header.Get(middleware.CSRFHeader)
	if token == "" {
		t.Fatal("missing csrf token")
	}
	cookie := ""
	for _, c := range get.Raw.Result().Cookies() {
		if c.Name == "pisigo_csrf" {
			cookie = c.Value
		}
	}
	res := pisigotest.Do(app, pisigotest.Request{Method: "POST", Path: "/form"})
	if res.Code != 403 {
		t.Fatalf("expected 403 got %d", res.Code)
	}
	res = pisigotest.Do(app, pisigotest.Request{
		Method: "POST",
		Path:   "/form",
		Headers: map[string]string{
			middleware.CSRFHeader: token,
			"Cookie":              "pisigo_csrf=" + cookie,
		},
	})
	if res.Code != 200 {
		t.Fatalf("csrf post=%d body=%s", res.Code, res.String())
	}
}

func TestCache(t *testing.T) {
	store := memory.NewStore()
	app := pisigo.Boot()
	app.Use(middleware.Cache(middleware.CacheConfig{Store: store, TTL: time.Minute}))
	hits := 0
	app.GET("/c", func(c *pisigo.Context) error {
		hits++
		return c.JSON(200, map[string]int{"n": hits})
	})
	r1 := pisigotest.GET(app, "/c")
	if r1.Header.Get("X-Cache") != "MISS" {
		t.Fatalf("miss=%q", r1.Header.Get("X-Cache"))
	}
	r2 := pisigotest.GET(app, "/c")
	if r2.Header.Get("X-Cache") != "HIT" {
		t.Fatalf("hit=%q body=%s", r2.Header.Get("X-Cache"), r2.String())
	}
	if hits != 1 {
		t.Fatalf("hits=%d", hits)
	}
}
