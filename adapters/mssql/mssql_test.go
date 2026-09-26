// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package mssql_test

import (
	"testing"

	"github.com/arnaudovproject/pisigo/db"
)

func TestMSSQLDSNShape(t *testing.T) {
	dsn, err := db.SQLConfig{Driver: "mssql", Host: "h", Port: 1433, User: "u", Password: "p", Database: "d"}.BuildDSN()
	if err != nil || dsn == "" {
		t.Fatal(err, dsn)
	}
}
