package pisigotest_test

import (
	"testing"

	"github.com/arnaudovproject/pisigo"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestHelpers(t *testing.T) {
	app := pisigo.Boot()
	app.POST("/echo", func(c *pisigo.Context) error {
		var m map[string]string
		if err := c.BindJSON(&m); err != nil {
			return err
		}
		return c.JSON(200, m)
	})
	res := pisigotest.POST(app, "/echo", map[string]string{"a": "b"})
	if res.Code != 200 {
		t.Fatalf("%d %s", res.Code, res.String())
	}
	var out map[string]string
	if err := res.JSON(&out); err != nil || out["a"] != "b" {
		t.Fatal(err, out)
	}
}
