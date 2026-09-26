// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNoRows     = sql.ErrNoRows
	ErrNotFound   = errors.New("db: not found")
	ErrInvalidDSN = errors.New("db: invalid dsn or config")
)

type Option func(*SQL)

func MaxOpenConns(n int) Option {
	return func(s *SQL) { s.maxOpen = n }
}

func MaxIdleConns(n int) Option {
	return func(s *SQL) { s.maxIdle = n }
}

func ConnMaxLifetime(d time.Duration) Option {
	return func(s *SQL) { s.maxLifetime = d }
}

func ConnMaxIdleTime(d time.Duration) Option {
	return func(s *SQL) { s.maxIdleTime = d }
}

type Result = sql.Result

type SQL struct {
	db          *sql.DB
	driver      string
	maxOpen     int
	maxIdle     int
	maxLifetime    time.Duration
	maxIdleTime time.Duration
}

func Open(driver, dsn string, opts ...Option) (*SQL, error) {
	if driver == "" || dsn == "" {
		return nil, ErrInvalidDSN
	}
	raw, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	s := &SQL{
		db:          raw,
		driver:      driver,
		maxOpen:     25,
		maxIdle:     5,
		maxLifetime:    time.Hour,
		maxIdleTime: 10 * time.Minute,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.applyPool()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		_ = raw.Close()
		return nil, err
	}
	return s, nil
}

func Wrap(raw *sql.DB, opts ...Option) *SQL {
	s := &SQL{
		db:          raw,
		maxOpen:     25,
		maxIdle:     5,
		maxLifetime:    time.Hour,
		maxIdleTime: 10 * time.Minute,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.applyPool()
	return s
}

func (s *SQL) applyPool() {
	if s.maxOpen > 0 {
		s.db.SetMaxOpenConns(s.maxOpen)
	}
	if s.maxIdle > 0 {
		s.db.SetMaxIdleConns(s.maxIdle)
	}
	if s.maxLifetime > 0 {
		s.db.SetConnMaxLifetime(s.maxLifetime)
	}
	if s.maxIdleTime > 0 {
		s.db.SetConnMaxIdleTime(s.maxIdleTime)
	}
}

func (s *SQL) DB() *sql.DB {
	return s.db
}

func (s *SQL) Driver() string {
	return s.driver
}

func (s *SQL) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQL) Close() error {
	return s.db.Close()
}

func (s *SQL) Exec(ctx context.Context, query string, args ...any) (Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

func (s *SQL) Query(ctx context.Context, query string, args ...any) (*Rows, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{rows: rows}, nil
}

func (s *SQL) QueryRow(ctx context.Context, query string, args ...any) *Row {
	return &Row{row: s.db.QueryRowContext(ctx, query, args...)}
}

func (s *SQL) Get(ctx context.Context, dest any, query string, args ...any) error {
	rows, err := s.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanOne(rows, dest)
}

func (s *SQL) Select(ctx context.Context, dest any, query string, args ...any) error {
	rows, err := s.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanAll(rows, dest)
}

func (s *SQL) Transaction(ctx context.Context, fn func(tx *Tx) error) error {
	return s.TransactionOptions(ctx, nil, fn)
}

func (s *SQL) TransactionOptions(ctx context.Context, opts *sql.TxOptions, fn func(tx *Tx) error) error {
	raw, err := s.db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	tx := &Tx{tx: raw}
	if err := fn(tx); err != nil {
		_ = raw.Rollback()
		return err
	}
	return raw.Commit()
}

type Rows struct {
	rows *sql.Rows
}

func (r *Rows) Next() bool              { return r.rows.Next() }
func (r *Rows) Err() error              { return r.rows.Err() }
func (r *Rows) Close() error            { return r.rows.Close() }
func (r *Rows) Columns() ([]string, error) { return r.rows.Columns() }
func (r *Rows) Scan(dest ...any) error  { return r.rows.Scan(dest...) }
func (r *Rows) Map() (map[string]any, error) {
	return scanMap(r)
}

type Row struct {
	row *sql.Row
}

func (r *Row) Scan(dest ...any) error {
	return r.row.Scan(dest...)
}

type Tx struct {
	tx *sql.Tx
}

func (t *Tx) Exec(ctx context.Context, query string, args ...any) (Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *Tx) Query(ctx context.Context, query string, args ...any) (*Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &Rows{rows: rows}, nil
}

func (t *Tx) QueryRow(ctx context.Context, query string, args ...any) *Row {
	return &Row{row: t.tx.QueryRowContext(ctx, query, args...)}
}

func (t *Tx) Get(ctx context.Context, dest any, query string, args ...any) error {
	rows, err := t.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanOne(rows, dest)
}

func (t *Tx) Select(ctx context.Context, dest any, query string, args ...any) error {
	rows, err := t.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanAll(rows, dest)
}

func (t *Tx) Rollback() error { return t.tx.Rollback() }
func (t *Tx) Commit() error   { return t.tx.Commit() }
