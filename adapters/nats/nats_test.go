// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package nats

import (
	"testing"

	"github.com/arnaudovproject/pisigo/pubsub"
	"github.com/arnaudovproject/pisigo/queue"
)

func TestInterfaceAssertions(t *testing.T) {
	var _ queue.Queue = (*Queue)(nil)
	var _ pubsub.PubSub = (*PubSub)(nil)
}
