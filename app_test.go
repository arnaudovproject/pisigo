// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigo_test

import (
	"testing"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/middleware"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestJSONRoute(t *testing.T) {
	app := pisigo.Boot()
	app.Use(middleware.Recover(), middleware.RequestID())
	app.GET("/hello", func(c *pisigo.Context) error {
		return c.JSON(200, map[string]string{"msg": "ok"})
	})

	res := pisigotest.GET(app, "/hello")
	if res.Code != 200 {
		t.Fatalf("status=%d body=%s", res.Code, res.String())
	}
	var body map[string]string
	if err := res.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["msg"] != "ok" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestHTTPError(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/missing", func(c *pisigo.Context) error {
		return pisigo.ErrNotFound
	})
	res := pisigotest.GET(app, "/missing")
	if res.Code != 404 {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestBindValidate(t *testing.T) {
	type In struct {
		Email string `json:"email" validate:"required,email"`
	}
	app := pisigo.Boot()
	app.POST("/in", func(c *pisigo.Context) error {
		var in In
		if err := c.Bind(&in); err != nil {
			return err
		}
		return c.JSON(200, in)
	})
	res := pisigotest.POST(app, "/in", map[string]string{"email": "bad"})
	if res.Code != 400 {
		t.Fatalf("expected 400 got %d body=%s", res.Code, res.String())
	}
	res = pisigotest.POST(app, "/in", map[string]string{"email": "a@b.com"})
	if res.Code != 200 {
		t.Fatalf("expected 200 got %d body=%s", res.Code, res.String())
	}
}

func TestNotFound(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/x", func(c *pisigo.Context) error { return c.String(200, "x") })
	res := pisigotest.GET(app, "/nope")
	if res.Code != 404 {
		t.Fatalf("status=%d", res.Code)
	}
}
