// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package redis

import (
	"testing"

	"github.com/arnaudovproject/pisigo/middleware"
	"github.com/arnaudovproject/pisigo/queue"
	"github.com/arnaudovproject/pisigo/store"
)

func TestInterfaceAssertions(t *testing.T) {
	var _ store.Store = (*Store)(nil)
	var _ queue.Queue = (*Queue)(nil)
	var _ middleware.RateLimitStore = (*RateLimitStore)(nil)
}
