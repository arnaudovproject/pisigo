package sse_test

import (
	"strings"
	"testing"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/sse"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestSSEEvent(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/sse", func(c *pisigo.Context) error {
		s, err := sse.Open(c)
		if err != nil {
			return err
		}
		if err := s.Event("msg", "line1\nline2"); err != nil {
			return err
		}
		return s.Comment("hi")
	})
	res := pisigotest.GET(app, "/sse")
	if res.Code != 200 {
		t.Fatalf("status=%d", res.Code)
	}
	body := res.String()
	if !strings.Contains(body, "event: msg") || !strings.Contains(body, "data: line1") || !strings.Contains(body, "data: line2") {
		t.Fatalf("body=%q", body)
	}
}
