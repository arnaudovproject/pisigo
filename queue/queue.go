// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package queue

import (
	"context"
	"errors"
	"time"
)

var (
	ErrClosed  = errors.New("queue: closed")
	ErrTimeout = errors.New("queue: timeout")
)

type Message struct {
	ID        string
	Topic     string
	Body      []byte
	Headers   map[string]string
	Timestamp time.Time
}

type Handler func(ctx context.Context, msg Message) error

type Queue interface {
	Publish(ctx context.Context, topic string, body []byte, headers map[string]string) error
	Subscribe(ctx context.Context, topic string, handler Handler) error
	Close() error
}
