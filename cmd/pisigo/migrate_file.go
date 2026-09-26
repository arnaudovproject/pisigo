// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func writeMigration(dir, name string) (string, error) {
	name = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "_"))
	filename := fmt.Sprintf("%s_%s.sql", time.Now().UTC().Format("20060102150405"), name)
	path := filepath.Join(dir, filename)
	body := "-- +migrate Up\n\n-- +migrate Down\n\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
