// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package circuit

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrOpen = errors.New("circuit: open")

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type Breaker struct {
	mu          sync.Mutex
	failures    int
	successes   int
	threshold   int
	successNeed int
	openUntil   time.Time
	cooldown    time.Duration
	state       State
	halfOpenInFlight bool
}

func New(threshold int, cooldown time.Duration) *Breaker {
	if threshold <= 0 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &Breaker{
		threshold:   threshold,
		successNeed: 1,
		cooldown:    cooldown,
		state:       StateClosed,
	}
}

func (b *Breaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case StateOpen:
		if time.Now().After(b.openUntil) {
			b.state = StateHalfOpen
			b.halfOpenInFlight = true
			return nil
		}
		return ErrOpen
	case StateHalfOpen:
		if b.halfOpenInFlight {
			return ErrOpen
		}
		b.halfOpenInFlight = true
		return nil
	default:
		return nil
	}
}

func (b *Breaker) Report(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err == nil {
		b.failures = 0
		if b.state == StateHalfOpen {
			b.successes++
			b.halfOpenInFlight = false
			if b.successes >= b.successNeed {
				b.state = StateClosed
				b.successes = 0
			}
		}
		return
	}
	b.failures++
	b.successes = 0
	b.halfOpenInFlight = false
	if b.failures >= b.threshold || b.state == StateHalfOpen {
		b.state = StateOpen
		b.openUntil = time.Now().Add(b.cooldown)
		b.failures = 0
	}
}

func (b *Breaker) Do(ctx context.Context, fn func(context.Context) error) error {
	if err := b.Allow(); err != nil {
		return err
	}
	err := fn(ctx)
	b.Report(err)
	return err
}

func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

type RetryConfig struct {
	Attempts int
	Delay    time.Duration
	Backoff  float64
}

func Retry(ctx context.Context, cfg RetryConfig, fn func(context.Context) error) error {
	if cfg.Attempts <= 0 {
		cfg.Attempts = 3
	}
	if cfg.Delay <= 0 {
		cfg.Delay = 100 * time.Millisecond
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = 2
	}
	var err error
	delay := cfg.Delay
	for i := 0; i < cfg.Attempts; i++ {
		if err = ctx.Err(); err != nil {
			return err
		}
		err = fn(ctx)
		if err == nil {
			return nil
		}
		if i == cfg.Attempts-1 {
			break
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		delay = time.Duration(float64(delay) * cfg.Backoff)
	}
	return err
}
