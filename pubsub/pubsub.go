package pubsub

import (
	"context"
	"errors"
	"time"
)

var ErrClosed = errors.New("pubsub: closed")

type Message struct {
	Subject   string
	Body      []byte
	Headers   map[string]string
	Timestamp time.Time
}

type Handler func(ctx context.Context, msg Message) error

type PubSub interface {
	Publish(ctx context.Context, subject string, body []byte, headers map[string]string) error
	Subscribe(ctx context.Context, subject string, handler Handler) error
	Request(ctx context.Context, subject string, body []byte, timeout time.Duration) (*Message, error)
	Close() error
}
