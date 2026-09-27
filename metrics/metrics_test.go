package metrics_test

import (
	"strings"
	"testing"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/metrics"
	"github.com/arnaudovproject/pisigo/middleware"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestMetricsRegister(t *testing.T) {
	app := pisigo.Boot()
	m := metrics.New("test")
	app.Use(m.Middleware())
	m.Register(app, "/metrics")
	app.GET("/hello", func(c *pisigo.Context) error { return c.String(200, "ok") })

	if pisigotest.GET(app, "/hello").Code != 200 {
		t.Fatal("hello")
	}
	res := pisigotest.GET(app, "/metrics")
	if res.Code != 200 {
		t.Fatalf("metrics=%d", res.Code)
	}
	if !strings.Contains(res.String(), "http_requests_total") {
		t.Fatalf("body=%s", res.String())
	}
}

func TestMetricsRegisterProtected(t *testing.T) {
	app := pisigo.Boot()
	m := metrics.New("test")
	m.Register(app, "/metrics", middleware.BasicAuth(middleware.BasicAuthConfig{
		Validator: func(u, p string) bool { return u == "prom" && p == "secret" },
	}))
	res := pisigotest.GET(app, "/metrics")
	if res.Code != 401 {
		t.Fatalf("unauth=%d", res.Code)
	}
	res = pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/metrics",
		Headers: map[string]string{"Authorization": "Basic cHJvbTpzZWNyZXQ="}, // prom:secret
	})
	if res.Code != 200 {
		t.Fatalf("auth=%d", res.Code)
	}
}
