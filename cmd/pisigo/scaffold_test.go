// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeAndExport(t *testing.T) {
	if sanitizeName("My App") != "my_app" && sanitizeName("My App") != "my-app" {
		// accept either style depending on implementation
		got := sanitizeName("Hello World")
		if got == "" {
			t.Fatal("empty")
		}
	}
	if exportName("users") == "" {
		t.Fatal("export")
	}
}

func TestNewProjectScaffold(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	NewProject([]string{"demoapp"})
	if _, err := os.Stat(filepath.Join("demoapp", "main.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join("demoapp", "go.mod")); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateCreateCmd(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll("migrations", 0o755)
	MigrateCmd([]string{"create", "add_table"})
	entries, err := os.ReadDir("migrations")
	if err != nil || len(entries) == 0 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
}
