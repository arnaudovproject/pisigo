// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package redis

import (
	"context"
	"time"

	"github.com/arnaudovproject/pisigo/store"
	"github.com/arnaudovproject/pisigo/queue"
	goredis "github.com/redis/go-redis/v9"
)

func Open(cfg Config) (*Store, *Queue, error) {
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
		return nil, nil, err
	}
	s := &Store{client: client}
	q := &Queue{client: client, prefix: "pisigo:queue:"}
	return s, q, nil
}

var (
	_ store.Store = (*Store)(nil)
	_ queue.Queue = (*Queue)(nil)
)
