// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigo_test

import (
	"errors"
	"testing"

	"github.com/arnaudovproject/pisigo"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestHTTPErrorDetailsOmitted(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/e", func(c *pisigo.Context) error {
		return pisigo.ErrBadRequest
	})
	res := pisigotest.GET(app, "/e")
	if res.Code != 400 {
		t.Fatalf("status=%d", res.Code)
	}
	if stringsContains(res.String(), `"details"`) {
		t.Fatalf("details should be omitted: %s", res.String())
	}
}

func TestHTTPErrorWithDetails(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/e", func(c *pisigo.Context) error {
		return pisigo.NewHTTPError(400, "bad").WithDetails(map[string]any{"f": "x"})
	})
	res := pisigotest.GET(app, "/e")
	if !stringsContains(res.String(), `"details"`) {
		t.Fatalf("expected details: %s", res.String())
	}
}

func TestUnhandledError(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/e", func(c *pisigo.Context) error {
		return errors.New("boom")
	})
	res := pisigotest.GET(app, "/e")
	if res.Code != 500 {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestAsHTTPError(t *testing.T) {
	he, ok := pisigo.AsHTTPError(pisigo.ErrNotFound)
	if !ok || he.Status() != 404 {
		t.Fatal("AsHTTPError failed")
	}
	if _, ok := pisigo.AsHTTPError(errors.New("x")); ok {
		t.Fatal("expected false")
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
