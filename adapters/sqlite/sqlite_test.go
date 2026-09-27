package sqlite_test

import (
	"context"
	"testing"

	"github.com/arnaudovproject/pisigo/adapters/sqlite"
	"github.com/arnaudovproject/pisigo/db"
)

func TestSQLiteMemoryCRUD(t *testing.T) {
	sqlDB, err := sqlite.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	ctx := context.Background()
	_, err = sqlDB.Exec(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.Exec(ctx, `INSERT INTO users(name) VALUES(?)`, "bob")
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err := sqlDB.Get(ctx, &name, `SELECT name FROM users WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if name != "bob" {
		t.Fatalf("%q", name)
	}
	var names []string
	if err := sqlDB.Select(ctx, &names, `SELECT name FROM users`); err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 {
		t.Fatalf("%v", names)
	}
	err = sqlDB.Transaction(ctx, func(tx *db.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO users(name) VALUES(?)`, "ann")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
