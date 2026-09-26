// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package memory

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/arnaudovproject/pisigo/store"
)

type Store struct {
	mu    sync.RWMutex
	items map[string]item
}

type item struct {
	value  string
	expire time.Time
}

func NewStore() *Store {
	return &Store{items: make(map[string]item)}
}

func (s *Store) Get(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[key]
	if !ok {
		return "", store.ErrNotFound
	}
	if !it.expire.IsZero() && time.Now().After(it.expire) {
		delete(s.items, key)
		return "", store.ErrNotFound
	}
	return it.value, nil
}

func (s *Store) Set(_ context.Context, key string, value string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	it := item{value: value}
	if ttl > 0 {
		it.expire = time.Now().Add(ttl)
	}
	s.items[key] = it
	return nil
}

func (s *Store) Delete(_ context.Context, keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range keys {
		delete(s.items, key)
	}
	return nil
}

func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.Get(ctx, key)
	if err == store.ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) Incr(_ context.Context, key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[key]
	var n int64
	alive := ok && (it.expire.IsZero() || time.Now().Before(it.expire))
	if alive {
		n, _ = strconv.ParseInt(it.value, 10, 64)
	}
	n++
	s.items[key] = item{value: strconv.FormatInt(n, 10)}
	return n, nil
}

func (s *Store) Expire(_ context.Context, key string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[key]
	if !ok {
		return store.ErrNotFound
	}
	if !it.expire.IsZero() && time.Now().After(it.expire) {
		delete(s.items, key)
		return store.ErrNotFound
	}
	if ttl > 0 {
		it.expire = time.Now().Add(ttl)
	} else {
		it.expire = time.Time{}
	}
	s.items[key] = it
	return nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[string]item)
	return nil
}
