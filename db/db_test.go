// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package db_test

import (
	"testing"

	"github.com/arnaudovproject/pisigo/db"
)

func TestBuildDSN(t *testing.T) {
	cases := []struct {
		cfg  db.SQLConfig
		want string
	}{
		{db.SQLConfig{Driver: "postgres", User: "u", Password: "p", Host: "h", Port: 5432, Database: "d"}, "postgres://u:p@h:5432/d?sslmode=require"},
		{db.SQLConfig{Driver: "postgres", User: "u", Password: "p", Host: "h", Port: 5432, Database: "d", SSLMode: "disable"}, "postgres://u:p@h:5432/d?sslmode=disable"},
		{db.SQLConfig{Driver: "mysql", User: "u", Password: "p", Host: "h", Port: 3306, Database: "d"}, "u:p@tcp(h:3306)/d?charset=utf8mb4&parseTime=true"},
		{db.SQLConfig{Driver: "sqlite", Database: ":memory:"}, ":memory:"},
		{db.SQLConfig{Driver: "mssql", Host: "h", Port: 1433, Database: "d", User: "u", Password: "p"}, "sqlserver://h:1433?database=d&password=p&user+id=u"},
	}
	for _, tc := range cases {
		got, err := tc.cfg.BuildDSN()
		if err != nil {
			t.Fatalf("%v", err)
		}
		if got != tc.want {
			t.Fatalf("driver=%s\ngot  %s\nwant %s", tc.cfg.Driver, got, tc.want)
		}
	}
	if _, err := (db.SQLConfig{Driver: "nope"}).BuildDSN(); err == nil {
		t.Fatal("expected error")
	}
}
