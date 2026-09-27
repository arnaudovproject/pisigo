package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/arnaudovproject/pisigo"
)

type Stream struct {
	w http.ResponseWriter
	f http.Flusher
}

func Open(c *pisigo.Context) (*Stream, error) {
	f, ok := c.Response().(http.Flusher)
	if !ok {
		return nil, pisigo.NewHTTPError(500, "streaming unsupported")
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(200)
	return &Stream{w: c.Response(), f: f}, nil
}

func (s *Stream) Event(name string, data any) error {
	payload, err := encode(data)
	if err != nil {
		return err
	}
	if name != "" {
		if _, err := fmt.Fprintf(s.w, "event: %s\n", name); err != nil {
			return err
		}
	}
	for _, line := range strings.Split(payload, "\n") {
		if _, err := fmt.Fprintf(s.w, "data: %s\n", line); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(s.w, "\n"); err != nil {
		return err
	}
	s.f.Flush()
	return nil
}

func (s *Stream) Comment(text string) error {
	if _, err := fmt.Fprintf(s.w, ": %s\n\n", text); err != nil {
		return err
	}
	s.f.Flush()
	return nil
}

func (s *Stream) Retry(ms int) error {
	if _, err := fmt.Fprintf(s.w, "retry: %d\n\n", ms); err != nil {
		return err
	}
	s.f.Flush()
	return nil
}

func encode(data any) (string, error) {
	switch v := data.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
