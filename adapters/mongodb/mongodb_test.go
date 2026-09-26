// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package mongodb_test

import (
	"testing"
	"time"

	"github.com/arnaudovproject/pisigo/adapters/mongodb"
)

func TestOpenInvalid(t *testing.T) {
	_, err := mongodb.Open(mongodb.Config{})
	if err == nil {
		t.Fatal("expected invalid config error")
	}
}

func TestOpenUnreachable(t *testing.T) {
	_, err := mongodb.Open(mongodb.Config{
		URI:      "mongodb://127.0.0.1:1",
		Database: "test",
		Timeout:  200 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected connection error")
	}
}
