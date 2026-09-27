package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/arnaudovproject/pisigo/middleware"
	goredis "github.com/redis/go-redis/v9"
)

// Atomic INCR + PEXPIRE so a crash between commands cannot leave a key without TTL.
var rateLimitScript = goredis.NewScript(`
local n = redis.call("INCR", KEYS[1])
if n == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return n
`)

type RateLimitStore struct {
	client *goredis.Client
	prefix string
}

func NewRateLimitStore(client *goredis.Client, prefix string) *RateLimitStore {
	if prefix == "" {
		prefix = "pisigo:ratelimit:"
	}
	return &RateLimitStore{client: client, prefix: prefix}
}

func (s *RateLimitStore) Allow(key string, limit int, window time.Duration) (bool, error) {
	ctx := context.Background()
	rk := s.prefix + key
	ms := window.Milliseconds()
	if ms <= 0 {
		ms = 1
	}
	n, err := rateLimitScript.Run(ctx, s.client, []string{rk}, ms).Int64()
	if err != nil {
		return false, err
	}
	return n <= int64(limit), nil
}

func (s *RateLimitStore) Remaining(key string, limit int) (int, error) {
	ctx := context.Background()
	n, err := s.client.Get(ctx, s.prefix+key).Result()
	if err == goredis.Nil {
		return limit, nil
	}
	if err != nil {
		return 0, err
	}
	count, _ := strconv.Atoi(n)
	left := limit - count
	if left < 0 {
		left = 0
	}
	return left, nil
}

var _ middleware.RateLimitStore = (*RateLimitStore)(nil)
