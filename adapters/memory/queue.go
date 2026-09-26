// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package memory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arnaudovproject/pisigo/queue"
)

type Queue struct {
	mu      sync.RWMutex
	topics  map[string][]chan queue.Message
	closed  atomic.Bool
	counter atomic.Uint64
}

func NewQueue() *Queue {
	return &Queue{topics: make(map[string][]chan queue.Message)}
}

func (q *Queue) Publish(ctx context.Context, topic string, body []byte, headers map[string]string) error {
	if q.closed.Load() {
		return queue.ErrClosed
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	msg := queue.Message{
		ID:        fmt.Sprintf("%d", q.counter.Add(1)),
		Topic:     topic,
		Body:      append([]byte(nil), body...),
		Headers:   copyHeaders(headers),
		Timestamp: time.Now(),
	}
	q.mu.RLock()
	subs := append([]chan queue.Message(nil), q.topics[topic]...)
	q.mu.RUnlock()
	for _, ch := range subs {
		safeSendQueue(ch, msg)
	}
	return nil
}

func safeSendQueue(ch chan queue.Message, msg queue.Message) {
	defer func() { _ = recover() }()
	select {
	case ch <- msg:
	default:
	}
}

func (q *Queue) Subscribe(ctx context.Context, topic string, handler queue.Handler) error {
	if q.closed.Load() {
		return queue.ErrClosed
	}
	ch := make(chan queue.Message, 64)
	q.mu.Lock()
	q.topics[topic] = append(q.topics[topic], ch)
	q.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			q.remove(topic, ch)
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return queue.ErrClosed
			}
			if err := handler(ctx, msg); err != nil {
				return err
			}
		}
	}
}

func (q *Queue) Close() error {
	if q.closed.Swap(true) {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	for topic, chans := range q.topics {
		for _, ch := range chans {
			close(ch)
		}
		delete(q.topics, topic)
	}
	return nil
}

func (q *Queue) remove(topic string, target chan queue.Message) {
	q.mu.Lock()
	defer q.mu.Unlock()
	subs := q.topics[topic]
	for i, ch := range subs {
		if ch == target {
			q.topics[topic] = append(subs[:i], subs[i+1:]...)
			close(target)
			break
		}
	}
}
