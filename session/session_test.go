package session_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/session"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestSessionLifecycle(t *testing.T) {
	store := session.NewMemoryStore()
	app := pisigo.Boot()
	app.Use(session.Middleware(session.Config{Store: store, Cookie: "sid", Path: "/"}))
	app.GET("/set", func(c *pisigo.Context) error {
		s, _ := session.FromContext(c)
		s.Set("user", "bob")
		return c.NoContent()
	})
	app.GET("/get", func(c *pisigo.Context) error {
		s, _ := session.FromContext(c)
		v, _ := s.Get("user")
		return c.JSON(200, map[string]any{"user": v})
	})
	app.POST("/bye", func(c *pisigo.Context) error {
		s, _ := session.FromContext(c)
		s.Destroy(c)
		return c.NoContent()
	})

	set := pisigotest.GET(app, "/set")
	var cookie *http.Cookie
	for _, c := range set.Raw.Result().Cookies() {
		if c.Name == "sid" {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("missing session cookie")
	}
	get := pisigotest.Do(app, pisigotest.Request{
		Method:  "GET",
		Path:    "/get",
		Headers: map[string]string{"Cookie": "sid=" + cookie.Value},
	})
	if get.Code != 200 || !contains(get.String(), "bob") {
		t.Fatalf("get=%s", get.String())
	}
	bye := pisigotest.Do(app, pisigotest.Request{
		Method:  "POST",
		Path:    "/bye",
		Headers: map[string]string{"Cookie": "sid=" + cookie.Value},
	})
	if bye.Code != 204 {
		t.Fatalf("bye=%d", bye.Code)
	}
}

func TestMemoryStoreExpiry(t *testing.T) {
	s := session.NewMemoryStore()
	_ = s.Save("a", map[string]any{"x": 1}, 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	if _, ok := s.Get("a"); ok {
		t.Fatal("expected expired")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && indexOf(s, sub) >= 0))
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
