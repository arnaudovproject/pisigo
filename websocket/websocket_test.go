// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package websocket_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/arnaudovproject/pisigo"
	"github.com/arnaudovproject/pisigo/websocket"
	gws "github.com/gorilla/websocket"
)

func TestUpgradeRejectOrigin(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/ws", websocket.Handler(func(c *pisigo.Context, conn *websocket.Conn) error {
		return conn.WriteMessage(gws.TextMessage, []byte("ok"))
	}, websocket.Config{
		CheckOrigin: func(r *http.Request) bool { return false },
	}))

	server := httptest.NewServer(app.Handler())
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	_, _, err := gws.DefaultDialer.Dial(url, nil)
	if err == nil {
		t.Fatal("expected origin rejection")
	}
}

func TestUpgradeOK(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/ws", websocket.Handler(func(c *pisigo.Context, conn *websocket.Conn) error {
		return conn.WriteMessage(gws.TextMessage, []byte("ok"))
	}))

	server := httptest.NewServer(app.Handler())
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := gws.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, msg, err := conn.ReadMessage()
	if err != nil || string(msg) != "ok" {
		t.Fatalf("msg=%q err=%v", msg, err)
	}
}

func TestUpgradeRejectCrossOrigin(t *testing.T) {
	app := pisigo.Boot()
	app.GET("/ws", websocket.Handler(func(c *pisigo.Context, conn *websocket.Conn) error {
		return conn.WriteMessage(gws.TextMessage, []byte("ok"))
	}))

	server := httptest.NewServer(app.Handler())
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	_, _, err := gws.DefaultDialer.Dial(url, http.Header{
		"Origin": []string{"https://evil.example"},
	})
	if err == nil {
		t.Fatal("expected cross-origin rejection")
	}
}
