// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package health_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/health"
	pisigotest "github.com/arnaudovproject/pisigo/testing"
)

func TestHealthHandlers(t *testing.T) {
	app := pisigo.Boot()
	h := health.New()
	h.Live("proc", func(ctx context.Context) error { return nil })
	h.Ready("db", func(ctx context.Context) error { return errors.New("down") })
	h.Register(app, "/health")

	live := pisigotest.GET(app, "/health/live")
	if live.Code != 200 {
		t.Fatalf("live=%d %s", live.Code, live.String())
	}
	ready := pisigotest.GET(app, "/health/ready")
	if ready.Code != 503 {
		t.Fatalf("ready=%d %s", ready.Code, ready.String())
	}
}
