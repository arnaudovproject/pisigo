package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/middleware"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestTimeoutContextRace(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.Logger(), middleware.Timeout(30*time.Millisecond))
	app.GET("/race", func(c *pisigo.Context) error {
		time.Sleep(100 * time.Millisecond)
		for i := 0; i < 100; i++ {
			c.Set("k", i)
			_, _ = c.Get("k")
			_ = c.StatusCode()
			_ = c.Written()
			_ = c.Log()
			_ = c.Method()
			_ = c.Path()
			_ = c.JSON(200, map[string]int{"i": i})
		}
		return nil
	})
	for i := 0; i < 20; i++ {
		res := pisigotest.GET(app, "/race")
		if res.Code != 504 {
			t.Fatalf("iter %d status=%d", i, res.Code)
		}
	}
}

func TestTimeoutPartialCommitKeepsStatus(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.Timeout(50 * time.Millisecond))
	app.GET("/partial", func(c *pisigo.Context) error {
		_ = c.String(http.StatusOK, "partial")
		time.Sleep(200 * time.Millisecond)
		return c.String(http.StatusOK, " late")
	})
	res := pisigotest.GET(app, "/partial")
	// Headers already committed as 200 — Timeout cannot rewrite to 504.
	if res.Code != 200 {
		t.Fatalf("expected 200 after commit, got %d body=%q", res.Code, res.String())
	}
	if body := res.String(); body != "partial" && body != "partial late" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestResponseRecorderFlushCommitsOK(t *testing.T) {
	rec := httptest.NewRecorder()
	app := pisigo.Boot()
	app.GET("/flush", func(c *pisigo.Context) error {
		flusher, ok := c.Response().(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter is not a Flusher")
		}
		flusher.Flush()
		if !c.Written() {
			t.Fatal("Flush should commit status 200")
		}
		if c.StatusCode() != http.StatusOK {
			t.Fatalf("status=%d", c.StatusCode())
		}
		_, err := c.Response().Write([]byte("hi"))
		return err
	})
	req := httptest.NewRequest(http.MethodGet, "/flush", nil)
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("code=%d", rec.Code)
	}
	if rec.Body.String() != "hi" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestCORSPreflightRejectsUnknownOrigin(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins: []string{"https://app.example"},
	}))
	app.GET("/", func(c *pisigo.Context) error { return c.NoContent() })

	res := pisigotest.Do(app, pisigotest.Request{
		Method:  "OPTIONS",
		Path:    "/",
		Headers: map[string]string{"Origin": "https://evil.example"},
	})
	if res.Code != 204 {
		t.Fatalf("status=%d", res.Code)
	}
	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no ACAO, got %q", got)
	}
	if got := res.Header.Get("Access-Control-Allow-Methods"); got != "" {
		t.Fatalf("expected no Allow-Methods for rejected origin, got %q", got)
	}
}
