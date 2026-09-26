// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package migrate_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/arnaudovproject/pisigo/adapters/sqlite"
	"github.com/arnaudovproject/pisigo/migrate"
)

func TestMigrateUpDown(t *testing.T) {
	sqlDB, err := sqlite.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "001_init.sql")
	body := `-- +migrate Up
CREATE TABLE t (id INTEGER PRIMARY KEY);
-- +migrate Down
DROP TABLE t;
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	m := migrate.New(sqlDB, dir)
	ctx := context.Background()
	if err := m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(ctx, `INSERT INTO t(id) VALUES(1)`); err != nil {
		t.Fatal(err)
	}
	if err := m.Down(ctx, 1); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateCreate(t *testing.T) {
	dir := t.TempDir()
	path, err := migrate.Create(dir, "add users")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
