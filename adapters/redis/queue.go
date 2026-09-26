// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arnaudovproject/pisigo/queue"
	goredis "github.com/redis/go-redis/v9"
)

type Queue struct {
	client *goredis.Client
	prefix string
}

type QueueConfig struct {
	Addr     string
	Password string
	DB       int
	Prefix   string
	Client   *goredis.Client
}

func NewQueue(cfg QueueConfig) (*Queue, error) {
	client := cfg.Client
	if client == nil {
		client = goredis.NewClient(&goredis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "pisigo:queue:"
	}
	return &Queue{client: client, prefix: prefix}, nil
}

func (q *Queue) key(topic string) string {
	return q.prefix + topic
}

func (q *Queue) Publish(ctx context.Context, topic string, body []byte, headers map[string]string) error {
	msg := queue.Message{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Topic:     topic,
		Body:      body,
		Headers:   headers,
		Timestamp: time.Now(),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, q.key(topic), payload).Err()
}

func (q *Queue) Subscribe(ctx context.Context, topic string, handler queue.Handler) error {
	key := q.key(topic)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		result, err := q.client.BRPop(ctx, 2*time.Second, key).Result()
		if err == goredis.Nil {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if len(result) < 2 {
			continue
		}
		var msg queue.Message
		if err := json.Unmarshal([]byte(result[1]), &msg); err != nil {
			continue
		}
		if err := handler(ctx, msg); err != nil {
			return err
		}
	}
}

func (q *Queue) Close() error {
	return q.client.Close()
}
