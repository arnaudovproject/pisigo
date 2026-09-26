// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package memory

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arnaudovproject/pisigo/pubsub"
)

type PubSub struct {
	mu       sync.RWMutex
	subjects map[string][]chan pubsub.Message
	closed   atomic.Bool
}

func NewPubSub() *PubSub {
	return &PubSub{subjects: make(map[string][]chan pubsub.Message)}
}

func (p *PubSub) Publish(ctx context.Context, subject string, body []byte, headers map[string]string) error {
	if p.closed.Load() {
		return pubsub.ErrClosed
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	msg := pubsub.Message{
		Subject:   subject,
		Body:      append([]byte(nil), body...),
		Headers:   copyHeaders(headers),
		Timestamp: time.Now(),
	}
	p.mu.RLock()
	subs := append([]chan pubsub.Message(nil), p.subjects[subject]...)
	p.mu.RUnlock()
	for _, ch := range subs {
		safeSendPubSub(ch, msg)
	}
	return nil
}

func safeSendPubSub(ch chan pubsub.Message, msg pubsub.Message) {
	defer func() { _ = recover() }()
	select {
	case ch <- msg:
	default:
	}
}

func (p *PubSub) Subscribe(ctx context.Context, subject string, handler pubsub.Handler) error {
	if p.closed.Load() {
		return pubsub.ErrClosed
	}
	ch := make(chan pubsub.Message, 64)
	p.mu.Lock()
	p.subjects[subject] = append(p.subjects[subject], ch)
	p.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			p.remove(subject, ch)
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return pubsub.ErrClosed
			}
			if err := handler(ctx, msg); err != nil {
				return err
			}
		}
	}
}

func (p *PubSub) Request(ctx context.Context, subject string, body []byte, timeout time.Duration) (*pubsub.Message, error) {
	reply := subject + ".reply." + time.Now().Format("150405.000000000")
	ch := make(chan pubsub.Message, 1)

	p.mu.Lock()
	p.subjects[reply] = append(p.subjects[reply], ch)
	p.mu.Unlock()
	defer p.remove(reply, ch)

	headers := map[string]string{"reply": reply}
	if err := p.Publish(ctx, subject, body, headers); err != nil {
		return nil, err
	}

	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, context.DeadlineExceeded
	case msg := <-ch:
		return &msg, nil
	}
}

func (p *PubSub) Close() error {
	if p.closed.Swap(true) {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for subject, chans := range p.subjects {
		for _, ch := range chans {
			close(ch)
		}
		delete(p.subjects, subject)
	}
	return nil
}

func (p *PubSub) remove(subject string, target chan pubsub.Message) {
	p.mu.Lock()
	defer p.mu.Unlock()
	subs := p.subjects[subject]
	for i, ch := range subs {
		if ch == target {
			p.subjects[subject] = append(subs[:i], subs[i+1:]...)
			close(target)
			break
		}
	}
}
