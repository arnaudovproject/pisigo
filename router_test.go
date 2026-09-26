// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigo_test

import (
	"testing"

	"github.com/arnaudovproject/pisigo"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestNamedRouteReverse(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/users/{id}", func(c *pisigo.Context) error {
		return c.String(200, c.Param("id"))
	}).SetName("user")

	path, err := app.Reverse("user", "id", "42")
	if err != nil || path != "/users/42" {
		t.Fatalf("path=%q err=%v", path, err)
	}
	if _, err := app.Reverse("missing"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := app.Reverse("user"); err == nil {
		t.Fatal("expected missing params error")
	}

	res := pisigotest.GET(app, "/users/42")
	if res.Code != 200 || res.String() != "42" {
		t.Fatalf("got %d %q", res.Code, res.String())
	}
}

func TestGroupNested(t *testing.T) {
	app := pisigo.Boot()
	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.GET("/ping", func(c *pisigo.Context) error {
		return c.String(200, "pong")
	})
	v1.GET("items", func(c *pisigo.Context) error {
		return c.String(200, "items")
	})

	res := pisigotest.GET(app, "/api/v1/ping")
	if res.Code != 200 || res.String() != "pong" {
		t.Fatalf("got %d %q", res.Code, res.String())
	}
	res = pisigotest.GET(app, "/api/v1/items")
	if res.Code != 200 || res.String() != "items" {
		t.Fatalf("got %d %q", res.Code, res.String())
	}
}

func TestMethodNotAllowed(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/only-get", func(c *pisigo.Context) error {
		return c.String(200, "ok")
	})
	res := pisigotest.Do(app, pisigotest.Request{Method: "POST", Path: "/only-get"})
	if res.Code != 405 {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestANY(t *testing.T) {
	app := pisigo.Boot()
	app.ANY("/any", func(c *pisigo.Context) error {
		return c.String(200, c.Method())
	})
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		res := pisigotest.Do(app, pisigotest.Request{Method: m, Path: "/any"})
		if res.Code != 200 || res.String() != m {
			t.Fatalf("%s => %d %q", m, res.Code, res.String())
		}
	}
}

func TestGroupMiddleware(t *testing.T) {
	app := pisigo.Boot()
	g := app.Group("/g", func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			c.Set("mw", "1")
			return next(c)
		}
	})
	g.GET("/x", func(c *pisigo.Context) error {
		v, _ := c.Get("mw")
		return c.JSON(200, map[string]any{"mw": v})
	})
	res := pisigotest.GET(app, "/g/x")
	if res.Code != 200 {
		t.Fatalf("status=%d", res.Code)
	}
}
