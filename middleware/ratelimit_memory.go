package middleware

import (
	"sync"
	"time"
)

type memoryRateLimitStore struct {
	mu       sync.Mutex
	data     map[string]*rateBucket
	ops      uint64
	gcEvery  uint64
}

type rateBucket struct {
	count int
	reset time.Time
}

func newMemoryRateLimitStore() *memoryRateLimitStore {
	return &memoryRateLimitStore{
		data:    make(map[string]*rateBucket),
		gcEvery: 256,
	}
}

func (s *memoryRateLimitStore) Allow(key string, limit int, window time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.ops++
	if s.gcEvery > 0 && s.ops%s.gcEvery == 0 {
		s.gcLocked(now)
	}
	b, ok := s.data[key]
	if !ok || now.After(b.reset) {
		s.data[key] = &rateBucket{count: 1, reset: now.Add(window)}
		return true, nil
	}
	if b.count >= limit {
		return false, nil
	}
	b.count++
	return true, nil
}

func (s *memoryRateLimitStore) gcLocked(now time.Time) {
	for k, b := range s.data {
		if now.After(b.reset) {
			delete(s.data, k)
		}
	}
}
