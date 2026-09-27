package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arnaudovproject/pisigo/config"
)

func TestLoadEnvFileAndHelpers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\nPORT=8080\nDEBUG=true\nTIMEOUT=2s\nNAME=\"pisigo\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Get("PORT", "") != "8080" {
		t.Fatal(cfg.Get("PORT", ""))
	}
	if cfg.Int("PORT", 0) != 8080 {
		t.Fatal(cfg.Int("PORT", 0))
	}
	if !cfg.Bool("DEBUG", false) {
		t.Fatal("bool")
	}
	if cfg.Duration("TIMEOUT", 0) != 2*time.Second {
		t.Fatal(cfg.Duration("TIMEOUT", 0))
	}
	if cfg.Get("NAME", "") != "pisigo" {
		t.Fatal(cfg.Get("NAME", ""))
	}
}

func TestLoadJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.json")
	if err := os.WriteFile(path, []byte(`{"db":{"host":"localhost","port":5432}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadJSONFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Get("db.host", "") != "localhost" {
		t.Fatal(cfg)
	}
	if cfg.Int("db.port", 0) != 5432 {
		t.Fatal(cfg.Int("db.port", 0))
	}
}

func TestLoadEnv(t *testing.T) {
	t.Setenv("PISIGO_TEST_X", "1")
	cfg := config.LoadEnv("PISIGO_TEST_X", "MISSING")
	if cfg.Get("PISIGO_TEST_X", "") != "1" {
		t.Fatal(cfg)
	}
}
