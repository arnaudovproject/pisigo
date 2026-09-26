// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package mysql

import (
	"fmt"

	"github.com/arnaudovproject/pisigo/db"
	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Params   map[string]string
	DSN      string
}

func Open(dsn string, opts ...db.Option) (*db.SQL, error) {
	return db.Open("mysql", dsn, opts...)
}

func OpenConfig(cfg Config, opts ...db.Option) (*db.SQL, error) {
	dsn := cfg.DSN
	if dsn == "" {
		built, err := db.SQLConfig{
			Driver:   "mysql",
			Host:     cfg.Host,
			Port:     cfg.Port,
			User:     cfg.User,
			Password: cfg.Password,
			Database: cfg.Database,
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
		panic(fmt.Errorf("mysql: %w", err))
	}
	return s
}
