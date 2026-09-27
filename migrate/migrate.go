package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/arnaudovproject/pisigo/db"
)

type Migrator struct {
	db   *db.SQL
	dir  string
	table string
}

func New(sqlDB *db.SQL, dir string) *Migrator {
	return &Migrator{db: sqlDB, dir: dir, table: "pisigo_migrations"}
}

func (m *Migrator) Ensure(ctx context.Context) error {
	_, err := m.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+m.table+` (
		id TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`)
	return err
}

func (m *Migrator) Up(ctx context.Context) error {
	if err := m.Ensure(ctx); err != nil {
		return err
	}
	files, err := m.files()
	if err != nil {
		return err
	}
	applied, err := m.applied(ctx)
	if err != nil {
		return err
	}
	for _, file := range files {
		name := filepath.Base(file)
		if applied[name] {
			continue
		}
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		up := extractSection(string(sqlBytes), "up")
		if up == "" {
			up = string(sqlBytes)
		}
		err = m.db.Transaction(ctx, func(tx *db.Tx) error {
			if _, err := tx.Exec(ctx, up); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, m.ph(`INSERT INTO `+m.table+`(id, applied_at) VALUES(?, ?)`), name, time.Now().UTC().Format(time.RFC3339))
			return err
		})
		if err != nil {
			return fmt.Errorf("migrate up %s: %w", name, err)
		}
	}
	return nil
}

func (m *Migrator) Down(ctx context.Context, steps int) error {
	if err := m.Ensure(ctx); err != nil {
		return err
	}
	if steps <= 0 {
		steps = 1
	}
	var ids []string
	if err := m.db.Select(ctx, &ids, `SELECT id FROM `+m.table+` ORDER BY applied_at DESC`); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if steps > len(ids) {
		steps = len(ids)
	}
	for i := 0; i < steps; i++ {
		name := ids[i]
		file := filepath.Join(m.dir, name)
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		down := extractSection(string(sqlBytes), "down")
		err = m.db.Transaction(ctx, func(tx *db.Tx) error {
			if down != "" {
				if _, err := tx.Exec(ctx, down); err != nil {
					return err
				}
			}
			_, err := tx.Exec(ctx, m.ph(`DELETE FROM `+m.table+` WHERE id = ?`), name)
			return err
		})
		if err != nil {
			return fmt.Errorf("migrate down %s: %w", name, err)
		}
	}
	return nil
}

func (m *Migrator) ph(query string) string {
	d := strings.ToLower(m.db.Driver())
	if d != "pgx" && d != "postgres" && d != "postgresql" {
		return query
	}
	var b strings.Builder
	n := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			b.WriteByte('$')
			b.WriteString(fmt.Sprintf("%d", n))
			n++
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

func (m *Migrator) files() ([]string, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".sql") {
			files = append(files, filepath.Join(m.dir, name))
		}
	}
	sort.Strings(files)
	return files, nil
}

func (m *Migrator) applied(ctx context.Context) (map[string]bool, error) {
	var ids []string
	if err := m.db.Select(ctx, &ids, `SELECT id FROM `+m.table); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

func extractSection(content, section string) string {
	marker := "-- +migrate " + section
	idx := strings.Index(strings.ToLower(content), strings.ToLower(marker))
	if idx < 0 {
		if section == "up" {
			return content
		}
		return ""
	}
	rest := content[idx+len(marker):]
	next := strings.Index(strings.ToLower(rest), "-- +migrate ")
	if next >= 0 {
		rest = rest[:next]
	}
	return strings.TrimSpace(rest)
}

func Create(dir, name string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s_%s.sql", time.Now().UTC().Format("20060102150405"), sanitize(name))
	path := filepath.Join(dir, filename)
	body := "-- +migrate Up\n\n-- +migrate Down\n\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func sanitize(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "_")
	return name
}
