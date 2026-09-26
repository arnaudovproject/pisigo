// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/arnaudovproject/pisigo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	requests *prometheus.CounterVec
	latency  *prometheus.HistogramVec
	registry *prometheus.Registry
}

func New(namespace string) *Metrics {
	if namespace == "" {
		namespace = "pisigo"
	}
	reg := prometheus.NewRegistry()
	m := &Metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests",
		}, []string{"method", "path", "status"}),
		latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "path"}),
		registry: reg,
	}
	reg.MustRegister(m.requests, m.latency)
	return m
}

func (m *Metrics) Middleware() pisigo.Middleware {
	return func(next pisigo.HandlerFunc) pisigo.HandlerFunc {
		return func(c *pisigo.Context) error {
			start := time.Now()
			err := next(c)
			path := c.Path()
			if pattern := c.Request().Pattern; pattern != "" {
				// Prefer ServeMux pattern (e.g. "GET /users/{id}") to avoid high cardinality.
				if i := len(pattern); i > 0 {
					for j := 0; j < len(pattern); j++ {
						if pattern[j] == ' ' {
							path = pattern[j+1:]
							break
						}
					}
				}
			}
			m.requests.WithLabelValues(c.Method(), path, strconv.Itoa(c.StatusCode())).Inc()
			m.latency.WithLabelValues(c.Method(), path).Observe(time.Since(start).Seconds())
			return err
		}
	}
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Register(app *pisigo.App, path string, protect ...pisigo.Middleware) {
	if path == "" {
		path = "/metrics"
	}
	handler := pisigo.HandlerFunc(func(c *pisigo.Context) error {
		m.Handler().ServeHTTP(c.Response(), c.Request())
		return nil
	})
	for i := len(protect) - 1; i >= 0; i-- {
		handler = protect[i](handler)
	}
	app.GET(path, handler)
}
