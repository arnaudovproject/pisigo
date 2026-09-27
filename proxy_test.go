package pisigo

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientIPFromForwardedRightmost(t *testing.T) {
	_, trustedNet, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	trusted := []*net.IPNet{trustedNet}

	got := clientIPFromForwarded("203.0.113.9, 10.0.0.5", trusted)
	if got != "203.0.113.9" {
		t.Fatalf("got %q want client", got)
	}

	// Spoofed leftmost must not win when right hop is trusted proxy.
	got = clientIPFromForwarded("198.51.100.1, 203.0.113.50, 10.0.0.2", trusted)
	if got != "203.0.113.50" {
		t.Fatalf("got %q want real client", got)
	}
}

func TestTrustedProxyUsesForwarded(t *testing.T) {
	app := Boot()
	if err := app.SetTrustedProxies("192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	app.GET("/ip", func(c *Context) error {
		return c.String(200, c.IP())
	})

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.7, 192.0.2.1")
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Body.String() != "198.51.100.7" {
		t.Fatalf("ip=%q", rec.Body.String())
	}
}

func TestDownloadFilenameSanitized(t *testing.T) {
	rec := httptest.NewRecorder()
	c := &Context{writer: rec, request: httptest.NewRequest(http.MethodGet, "/", nil)}
	_ = c.Download("README.md", "report\"\r\nX-Injected: yes.pdf")
	cd := rec.Header().Get("Content-Disposition")
	if strings.ContainsAny(cd, "\r\n") {
		t.Fatalf("CRLF in header: %q", cd)
	}
	if strings.Count(cd, `"`) != 2 {
		t.Fatalf("unexpected quoting: %q", cd)
	}
	if !strings.Contains(cd, "reportX-Injected: yes.pdf") {
		t.Fatalf("filename=%q", cd)
	}
}
