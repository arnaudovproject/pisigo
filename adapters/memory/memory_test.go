package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/arnaudovproject/pisigo/adapters/memory"
	"github.com/arnaudovproject/pisigo/pubsub"
	"github.com/arnaudovproject/pisigo/queue"
	"github.com/arnaudovproject/pisigo/store"
)

func TestStoreBasics(t *testing.T) {
	s := memory.NewStore()
	ctx := context.Background()
	if err := s.Set(ctx, "k", "v", 0); err != nil {
		t.Fatal(err)
	}
	v, err := s.Get(ctx, "k")
	if err != nil || v != "v" {
		t.Fatalf("%q %v", v, err)
	}
	ok, _ := s.Exists(ctx, "k")
	if !ok {
		t.Fatal("exists")
	}
	n, err := s.Incr(ctx, "n")
	if err != nil || n != 1 {
		t.Fatalf("%d %v", n, err)
	}
	n, _ = s.Incr(ctx, "n")
	if n != 2 {
		t.Fatalf("%d", n)
	}
	_ = s.Delete(ctx, "k")
	if _, err := s.Get(ctx, "k"); err != store.ErrNotFound {
		t.Fatalf("err=%v", err)
	}
}

func TestStoreTTLAndIncrExpired(t *testing.T) {
	s := memory.NewStore()
	ctx := context.Background()
	_ = s.Set(ctx, "t", "1", 15*time.Millisecond)
	time.Sleep(25 * time.Millisecond)
	if _, err := s.Get(ctx, "t"); err != store.ErrNotFound {
		t.Fatal(err)
	}
	_ = s.Set(ctx, "x", "9", 15*time.Millisecond)
	time.Sleep(25 * time.Millisecond)
	n, err := s.Incr(ctx, "x")
	if err != nil || n != 1 {
		t.Fatalf("incr after expire: %d %v", n, err)
	}
	if _, err := s.Get(ctx, "x"); err != nil {
		t.Fatal(err)
	}
}

func TestQueuePubSub(t *testing.T) {
	q := memory.NewQueue()
	defer q.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var got queue.Message
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = q.Subscribe(ctx, "jobs", func(ctx context.Context, msg queue.Message) error {
			got = msg
			cancel()
			return nil
		})
	}()
	time.Sleep(10 * time.Millisecond)
	if err := q.Publish(context.Background(), "jobs", []byte("hi"), map[string]string{"a": "1"}); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if string(got.Body) != "hi" || got.Headers["a"] != "1" {
		t.Fatalf("%#v", got)
	}
}

func TestPubSubRequest(t *testing.T) {
	ps := memory.NewPubSub()
	defer ps.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = ps.Subscribe(ctx, "echo", func(ctx context.Context, msg pubsub.Message) error {
			return ps.Publish(ctx, msg.Headers["reply"], []byte("pong"), nil)
		})
	}()
	time.Sleep(10 * time.Millisecond)
	resp, err := ps.Request(context.Background(), "echo", []byte("ping"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.Body) != "pong" {
		t.Fatalf("%s", resp.Body)
	}
}

func TestClosedQueue(t *testing.T) {
	q := memory.NewQueue()
	_ = q.Close()
	if err := q.Publish(context.Background(), "t", nil, nil); err != queue.ErrClosed {
		t.Fatalf("err=%v", err)
	}
}
