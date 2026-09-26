// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package sqlite

import (
	"fmt"

	"github.com/arnaudovproject/pisigo/db"
	_ "modernc.org/sqlite"
)

type Config struct {
	Path   string
	Memory bool
	Params map[string]string
	DSN    string
}

func Open(dsn string, opts ...db.Option) (*db.SQL, error) {
	return db.Open("sqlite", dsn, opts...)
}

func OpenConfig(cfg Config, opts ...db.Option) (*db.SQL, error) {
	dsn := cfg.DSN
	if dsn == "" {
		if cfg.Memory {
			dsn = ":memory:"
		} else if cfg.Path != "" {
			dsn = cfg.Path
		} else {
			return nil, db.ErrInvalidDSN
		}
		if len(cfg.Params) > 0 {
			q := ""
			first := true
			for k, v := range cfg.Params {
				if first {
					q += "?"
					first = false
				} else {
					q += "&"
				}
				q += k + "=" + v
			}
			dsn += q
		}
	}
	return Open(dsn, opts...)
}

func MustOpen(dsn string, opts ...db.Option) *db.SQL {
	s, err := Open(dsn, opts...)
	if err != nil {
		panic(fmt.Errorf("sqlite: %w", err))
	}
	return s
}

func OpenMemory(opts ...db.Option) (*db.SQL, error) {
	opts = append([]db.Option{db.MaxOpenConns(1), db.MaxIdleConns(1)}, opts...)
	return OpenConfig(Config{Memory: true}, opts...)
}
