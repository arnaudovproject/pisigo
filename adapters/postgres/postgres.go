package postgres

import (
	"fmt"

	"github.com/arnaudovproject/pisigo/db"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	Params   map[string]string
	DSN      string
}

func Open(dsn string, opts ...db.Option) (*db.SQL, error) {
	return db.Open("pgx", dsn, opts...)
}

func OpenConfig(cfg Config, opts ...db.Option) (*db.SQL, error) {
	dsn := cfg.DSN
	if dsn == "" {
		built, err := db.SQLConfig{
			Driver:   "pgx",
			Host:     cfg.Host,
			Port:     cfg.Port,
			User:     cfg.User,
			Password: cfg.Password,
			Database: cfg.Database,
			SSLMode:  cfg.SSLMode,
			Params:   cfg.Params,
		}.BuildDSN()
		if err != nil {
			return nil, err
		}
		dsn = built
	}
	return Open(dsn, opts...)
}

func MustOpen(dsn string, opts ...db.Option) *db.SQL {
	s, err := Open(dsn, opts...)
	if err != nil {
		panic(fmt.Errorf("postgres: %w", err))
	}
	return s
}
