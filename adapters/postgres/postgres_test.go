package postgres_test

import (
	"testing"

	"github.com/arnaudovproject/pisigo/db"
)

func TestPostgresDSNShape(t *testing.T) {
	dsn, err := db.SQLConfig{
		Driver:   "pgx",
		Host:     "localhost",
		Port:     5432,
		User:     "u",
		Password: "p",
		Database: "d",
	}.BuildDSN()
	if err != nil {
		t.Fatal(err)
	}
	if dsn != "postgres://u:p@localhost:5432/d?sslmode=require" {
		t.Fatalf("%s", dsn)
	}
}
