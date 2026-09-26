// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package circuit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arnaudovproject/pisigo/circuit"
)

func TestBreakerOpensAndHalfOpen(t *testing.T) {
	b := circuit.New(2, 30*time.Millisecond)
	fail := errors.New("fail")
	_ = b.Do(context.Background(), func(ctx context.Context) error { return fail })
	_ = b.Do(context.Background(), func(ctx context.Context) error { return fail })
	if b.State() != circuit.StateOpen {
		t.Fatalf("state=%v", b.State())
	}
	if err := b.Allow(); err != circuit.ErrOpen {
		t.Fatalf("err=%v", err)
	}
	time.Sleep(40 * time.Millisecond)
	if err := b.Allow(); err != nil {
		t.Fatalf("half-open allow: %v", err)
	}
	if err := b.Allow(); err != circuit.ErrOpen {
		t.Fatalf("second probe should be blocked: %v", err)
	}
	b.Report(nil)
	if b.State() != circuit.StateClosed {
		t.Fatalf("expected closed got %v", b.State())
	}
}

func TestRetry(t *testing.T) {
	n := 0
	err := circuit.Retry(context.Background(), circuit.RetryConfig{Attempts: 3, Delay: time.Millisecond}, func(ctx context.Context) error {
		n++
		if n < 3 {
			return errors.New("x")
		}
		return nil
	})
	if err != nil || n != 3 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestRetryCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := circuit.Retry(ctx, circuit.RetryConfig{Attempts: 3, Delay: time.Millisecond}, func(ctx context.Context) error {
		return errors.New("x")
	})
	if err == nil {
		t.Fatal("expected cancel")
	}
}
