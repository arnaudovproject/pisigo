// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigo_test

import (
	"strings"
	"testing"

	"github.com/arnaudovproject/pisigo"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestContextStoreAndIP(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/ctx", func(c *pisigo.Context) error {
		c.Set("k", "v")
		if c.MustGet("k") != "v" {
			t.Fatal("MustGet failed")
		}
		return c.JSON(200, map[string]string{"ip": c.IP()})
	})
	res := pisigotest.GET(app, "/ctx")
	if res.Code != 200 {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestBodyLimit(t *testing.T) {
	app := pisigo.Boot()
	app.SetMaxBodyBytes(8)
	app.POST("/body", func(c *pisigo.Context) error {
		_, err := c.Body()
		return err
	})
	res := pisigotest.POST(app, "/body", map[string]string{"too": "large-body"})
	if res.Code != 413 {
		t.Fatalf("expected 413 got %d body=%s", res.Code, res.String())
	}
}

func TestResponseHelpers(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/str", func(c *pisigo.Context) error { return c.String(201, "hi") })
	app.GET("/html", func(c *pisigo.Context) error { return c.HTML(200, "<b>x</b>") })
	app.GET("/nc", func(c *pisigo.Context) error { return c.NoContent() })
	app.GET("/redir", func(c *pisigo.Context) error { return c.Redirect(302, "/str") })
	app.GET("/data", func(c *pisigo.Context) error {
		return c.Data(200, "text/plain", []byte("raw"))
	})

	if code := pisigotest.GET(app, "/str").Code; code != 201 {
		t.Fatalf("str=%d", code)
	}
	if body := pisigotest.GET(app, "/html").String(); !strings.Contains(body, "<b>x</b>") {
		t.Fatalf("html=%q", body)
	}
	if code := pisigotest.GET(app, "/nc").Code; code != 204 {
		t.Fatalf("nc=%d", code)
	}
	res := pisigotest.GET(app, "/redir")
	if res.Code != 302 {
		t.Fatalf("redir=%d", res.Code)
	}
	if pisigotest.GET(app, "/data").String() != "raw" {
		t.Fatal("data mismatch")
	}
}

func TestQueryHelpers(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/q", func(c *pisigo.Context) error {
		return c.JSON(200, map[string]any{
			"a": c.Query("a"),
			"b": c.QueryDefault("b", "def"),
			"n": c.QueryInt("n", 0),
		})
	})
	res := pisigotest.GET(app, "/q?a=1&n=7")
	var body map[string]any
	if err := res.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["a"] != "1" || body["b"] != "def" || body["n"] != float64(7) {
		t.Fatalf("%#v", body)
	}
}

func TestBearerToken(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/tok", func(c *pisigo.Context) error {
		return c.String(200, c.BearerToken())
	})
	res := pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/tok",
		Headers: map[string]string{"Authorization": "Bearer abc"},
	})
	if res.String() != "abc" {
		t.Fatalf("got %q", res.String())
	}
}

func TestTrustedProxy(t *testing.T) {
	app := pisigo.Boot()
	if err := app.SetTrustedProxies("127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	app.GET("/ip", func(c *pisigo.Context) error {
		return c.String(200, c.IP())
	})
	res := pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/ip",
		Headers: map[string]string{"X-Forwarded-For": "203.0.113.9"},
	})
	// httptest RemoteAddr is typically 192.0.2.1 — not trusted, so spoofed XFF is ignored.
	if res.String() == "203.0.113.9" {
		t.Fatal("untrusted peer must not honor X-Forwarded-For")
	}
}

func TestFormValueRespectsBodyLimit(t *testing.T) {
	app := pisigo.Boot()
	app.SetMaxBodyBytes(8)
	app.POST("/form", func(c *pisigo.Context) error {
		_ = c.FormValue("a")
		return c.NoContent()
	})
	res := pisigotest.Do(app, pisigotest.Request{
		Method:  "POST",
		Path:    "/form",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:    "a=" + strings.Repeat("x", 64),
	})
	// Oversized form parse fails closed for FormValue (empty); handler still 204.
	// Ensure we do not panic and body limit path is exercised.
	if res.Code != 204 && res.Code != 413 {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestFileTracksRealStatus(t *testing.T) {
	app := pisigo.Boot()
	var logged int
	app.Use(func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			err := next(c)
			logged = c.StatusCode()
			return err
		}
	})
	app.GET("/missing-file", func(c *pisigo.Context) error {
		return c.File("/definitely/does/not/exist-" + t.Name())
	})
	res := pisigotest.GET(app, "/missing-file")
	if res.Code != 404 {
		t.Fatalf("client status=%d", res.Code)
	}
	if logged != 404 {
		t.Fatalf("StatusCode after ServeFile=%d want 404", logged)
	}
}
