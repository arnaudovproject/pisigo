package pisigo

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// responseRecorder tracks the real HTTP status and bytes written so helpers
// that call into net/http directly (ServeFile, FileServer, Redirect) stay
// consistent with Logger and StatusCode().
type responseRecorder struct {
	http.ResponseWriter
	mu      sync.Mutex
	status  int
	written bool
	size    int64
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

func (w *responseRecorder) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writeHeaderLocked(code)
}

func (w *responseRecorder) writeHeaderLocked(code int) {
	if w.written {
		return
	}
	if code == 0 {
		code = http.StatusOK
	}
	w.status = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseRecorder) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.written {
		w.writeHeaderLocked(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += int64(n)
	return n, err
}

func (w *responseRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseRecorder) Flush() {
	w.mu.Lock()
	if !w.written {
		w.writeHeaderLocked(http.StatusOK)
	}
	w.mu.Unlock()
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("pisigo: ResponseWriter does not support hijacking")
	}
	return h.Hijack()
}

func (w *responseRecorder) Push(target string, opts *http.PushOptions) error {
	p, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return p.Push(target, opts)
}

func (w *responseRecorder) snapshot() (status int, written bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status, w.written
}

func (w *responseRecorder) mark(status int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.written = true
	if status != 0 {
		w.status = status
	}
}

func findRecorder(w http.ResponseWriter) *responseRecorder {
	for w != nil {
		switch v := w.(type) {
		case *responseRecorder:
			return v
		case interface{ Unwrap() http.ResponseWriter }:
			w = v.Unwrap()
		default:
			return nil
		}
	}
	return nil
}
