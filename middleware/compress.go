// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/arnaudovproject/pisigo"
)

var gzipPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

type gzipWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	return g.writer.Write(b)
}

func (g *gzipWriter) Flush() {
	_ = g.writer.Flush()
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func Compress() pisigo.Middleware {
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			if !strings.Contains(c.HeaderGet("Accept-Encoding"), "gzip") {
				return next(c)
			}
			gz := gzipPool.Get().(*gzip.Writer)
			gz.Reset(c.Response())
			defer func() {
				_ = gz.Close()
				gzipPool.Put(gz)
			}()
			c.Header("Content-Encoding", "gzip")
			c.Header("Vary", "Accept-Encoding")
			c.Response().Header().Del("Content-Length")
			c.SetWriter(&gzipWriter{ResponseWriter: c.Response(), writer: gz})
			return next(c)
		}
	}
}
