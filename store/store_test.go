// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package store_test

import (
	"errors"
	"testing"

	"github.com/arnaudovproject/pisigo/store"
)

func TestSentinels(t *testing.T) {
	if !errors.Is(store.ErrNotFound, store.ErrNotFound) {
		t.Fatal()
	}
}
