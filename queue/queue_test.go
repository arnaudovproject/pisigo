package queue_test

import (
	"testing"

	"github.com/arnaudovproject/pisigo/queue"
)

func TestSentinels(t *testing.T) {
	if queue.ErrClosed == nil {
		t.Fatal()
	}
}
