// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package auth_test

import (
	"testing"
	"time"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/auth"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestJWTRoundTrip(t *testing.T) {
	cfg := auth.DefaultJWTConfig("secret-key-at-least-32-bytes!!")
	cfg.Issuer = "pisigo"
	cfg.Audience = "api"
	tok, err := auth.Issue(cfg, "u1", []string{"admin"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.Parse(cfg, tok)
	if err != nil || claims.UserID != "u1" {
		t.Fatalf("claims=%v err=%v", claims, err)
	}

	bad := cfg
	bad.Audience = "other"
	if _, err := auth.Parse(bad, tok); err == nil {
		t.Fatal("expected audience failure")
	}
}

func TestJWTMiddlewareAndRoles(t *testing.T) {
	cfg := auth.DefaultJWTConfig("secret-key-at-least-32-bytes!!")
	tok, _ := auth.Issue(cfg, "u1", []string{"editor"}, nil)

	app := pisigo.Boot()
	app.Use(auth.JWT(cfg), auth.RequireRoles("admin", "editor"))
	app.GET("/me", func(c *pisigo.Context) error {
		claims, ok := auth.ClaimsFromContext(c)
		if !ok {
			return pisigo.ErrUnauthorized
		}
		return c.JSON(200, map[string]string{"uid": claims.UserID})
	})

	res := pisigotest.GET(app, "/me")
	if res.Code != 401 {
		t.Fatalf("unauth=%d", res.Code)
	}
	res = pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/me",
		Headers: map[string]string{"Authorization": "Bearer " + tok},
	})
	if res.Code != 200 {
		t.Fatalf("status=%d body=%s", res.Code, res.String())
	}
}

func TestAPIKey(t *testing.T) {
	app := pisigo.Boot()
	app.Use(auth.APIKey("X-API-Key", func(k string) bool { return k == "ok" }))
	app.GET("/", func(c *pisigo.Context) error { return c.NoContent() })
	if pisigotest.GET(app, "/").Code != 401 {
		t.Fatal("expected 401")
	}
	res := pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/",
		Headers: map[string]string{"X-API-Key": "ok"},
	})
	if res.Code != 204 {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestExpiredToken(t *testing.T) {
	cfg := auth.DefaultJWTConfig("secret-key-at-least-32-bytes!!")
	cfg.TTL = -time.Hour
	tok, err := auth.Issue(cfg, "u", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Parse(cfg, tok); err == nil {
		t.Fatal("expected expired")
	}
}
