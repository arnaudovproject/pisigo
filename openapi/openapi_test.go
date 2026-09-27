package openapi_test

import (
	"strings"
	"testing"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/openapi"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestOpenAPIRegister(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/users", func(c *pisigo.Context) error { return c.NoContent() }).SetName("listUsers")
	openapi.Register(app, "/openapi.json", "Demo", "1.0.0")

	res := pisigotest.GET(app, "/openapi.json")
	if res.Code != 200 || !strings.Contains(res.String(), "listUsers") {
		t.Fatalf("%s", res.String())
	}
	docs := pisigotest.GET(app, "/docs")
	if docs.Code != 200 || !strings.Contains(docs.String(), "swagger-ui") {
		t.Fatalf("%s", docs.String())
	}
	raw, err := openapi.JSON(app, "Demo", "1.0.0")
	if err != nil || !strings.Contains(string(raw), "openapi") {
		t.Fatal(err)
	}
}
