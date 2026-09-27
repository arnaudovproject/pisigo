package nats

import (
	"context"
	"time"

	"github.com/arnaudovproject/pisigo/pubsub"
	natslib "github.com/nats-io/nats.go"
)

type Config struct {
	URL  string
	Name string
	Conn *natslib.Conn
}

type PubSub struct {
	conn *natslib.Conn
}

func NewPubSub(cfg Config) (*PubSub, error) {
	conn := cfg.Conn
	if conn == nil {
		opts := []natslib.Option{}
		if cfg.Name != "" {
			opts = append(opts, natslib.Name(cfg.Name))
		}
		url := cfg.URL
		if url == "" {
			url = natslib.DefaultURL
		}
		var err error
		conn, err = natslib.Connect(url, opts...)
		if err != nil {
			return nil, err
		}
	}
	return &PubSub{conn: conn}, nil
}

func (p *PubSub) Conn() *natslib.Conn {
	return p.conn
}

func (p *PubSub) Publish(_ context.Context, subject string, body []byte, headers map[string]string) error {
	msg := &natslib.Msg{Subject: subject, Data: body}
	if len(headers) > 0 {
		msg.Header = natslib.Header{}
		for k, v := range headers {
			msg.Header.Set(k, v)
		}
	}
	return p.conn.PublishMsg(msg)
}

func (p *PubSub) Subscribe(ctx context.Context, subject string, handler pubsub.Handler) error {
	ch := make(chan *natslib.Msg, 64)
	sub, err := p.conn.ChanSubscribe(subject, ch)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return pubsub.ErrClosed
			}
			headers := map[string]string{}
			for k, values := range msg.Header {
				if len(values) > 0 {
					headers[k] = values[0]
				}
			}
			if err := handler(ctx, pubsub.Message{
				Subject:   msg.Subject,
				Body:      msg.Data,
				Headers:   headers,
				Timestamp: time.Now(),
			}); err != nil {
				return err
			}
		}
	}
}

func (p *PubSub) Request(ctx context.Context, subject string, body []byte, timeout time.Duration) (*pubsub.Message, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	msg, err := p.conn.RequestWithContext(ctx, subject, body)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{}
	for k, values := range msg.Header {
		if len(values) > 0 {
			headers[k] = values[0]
		}
	}
	return &pubsub.Message{
		Subject:   msg.Subject,
		Body:      msg.Data,
		Headers:   headers,
		Timestamp: time.Now(),
	}, nil
}

func (p *PubSub) Close() error {
	p.conn.Close()
	return nil
}
