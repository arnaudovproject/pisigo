// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package websocket

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/arnaudovproject/pisigo"
	"github.com/gorilla/websocket"
)

var defaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     sameOrigin,
}

type Conn = websocket.Conn

type Config struct {
	CheckOrigin func(r *http.Request) bool
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Non-browser clients typically omit Origin.
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func Upgrade(c *pisigo.Context, cfg ...Config) (*Conn, error) {
	upgrader := defaultUpgrader
	if len(cfg) > 0 && cfg[0].CheckOrigin != nil {
		upgrader.CheckOrigin = cfg[0].CheckOrigin
	}
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func Handler(fn func(c *pisigo.Context, conn *Conn) error, cfg ...Config) pisigo.HandlerFunc {
	return func(c *pisigo.Context) error {
		conn, err := Upgrade(c, cfg...)
		if err != nil {
			return err
		}
		defer conn.Close()
		return fn(c, conn)
	}
}
