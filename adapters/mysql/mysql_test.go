// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package mysql_test

import (
	"testing"

	"github.com/arnaudovproject/pisigo/db"
)

func TestMySQLDSNShape(t *testing.T) {
	dsn, err := db.SQLConfig{Driver: "mysql", Host: "h", Port: 3306, User: "u", Password: "p", Database: "d"}.BuildDSN()
	if err != nil || dsn == "" {
		t.Fatal(err, dsn)
	}
}
