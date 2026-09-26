// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pubsub_test

import (
	"testing"

	"github.com/arnaudovproject/pisigo/pubsub"
)

func TestSentinels(t *testing.T) {
	if pubsub.ErrClosed == nil {
		t.Fatal()
	}
}
