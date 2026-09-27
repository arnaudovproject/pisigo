package otelx_test

import (
	"context"
	"testing"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/otelx"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestOtelSetupAndMiddleware(t *testing.T) {
	shutdown, err := otelx.Setup("pisigo-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	app := pisigo.Boot()
	app.Use(otelx.Middleware("pisigo-test"))
	app.GET("/", func(c *pisigo.Context) error { return c.String(200, "ok") })
	res := pisigotest.GET(app, "/")
	if res.Code != 200 {
		t.Fatalf("status=%d", res.Code)
	}
}
