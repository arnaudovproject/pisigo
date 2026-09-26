// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/arnaudovproject/pisigo/queue"
	natslib "github.com/nats-io/nats.go"
)

type Queue struct {
	conn   *natslib.Conn
	js     natslib.JetStreamContext
	stream string
}

type QueueConfig struct {
	URL    string
	Name   string
	Stream string
	Conn   *natslib.Conn
}

func NewQueue(cfg QueueConfig) (*Queue, error) {
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
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, err
	}
	stream := cfg.Stream
	if stream == "" {
		stream = "PISIGO"
	}
	_, err = js.StreamInfo(stream)
	if err != nil {
		_, err = js.AddStream(&natslib.StreamConfig{
			Name:     stream,
			Subjects: []string{stream + ".>"},
		})
		if err != nil {
			conn.Close()
			return nil, err
		}
	}
	return &Queue{conn: conn, js: js, stream: stream}, nil
}

func (q *Queue) subject(topic string) string {
	return q.stream + "." + topic
}

func (q *Queue) Publish(ctx context.Context, topic string, body []byte, headers map[string]string) error {
	msg := &natslib.Msg{Subject: q.subject(topic), Data: body}
	if len(headers) > 0 {
		msg.Header = natslib.Header{}
		for k, v := range headers {
			msg.Header.Set(k, v)
		}
	}
	_, err := q.js.PublishMsg(msg, natslib.Context(ctx))
	return err
}

func (q *Queue) Subscribe(ctx context.Context, topic string, handler queue.Handler) error {
	ch := make(chan *natslib.Msg, 64)
	sub, err := q.js.ChanSubscribe(q.subject(topic), ch)
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
				return queue.ErrClosed
			}
			headers := map[string]string{}
			for k, values := range msg.Header {
				if len(values) > 0 {
					headers[k] = values[0]
				}
			}
			qm := queue.Message{
				ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
				Topic:     topic,
				Body:      msg.Data,
				Headers:   headers,
				Timestamp: time.Now(),
			}
			if err := handler(ctx, qm); err != nil {
				return err
			}
			_ = msg.Ack()
		}
	}
}

func (q *Queue) Close() error {
	q.conn.Close()
	return nil
}
