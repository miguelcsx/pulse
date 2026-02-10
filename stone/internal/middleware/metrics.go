package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pulse",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests processed, partitioned by status code, method, and path.",
		},
		[]string{"code", "method", "path"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "pulse",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request latency distribution in seconds.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	httpActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "pulse",
			Subsystem: "http",
			Name:      "active_connections",
			Help:      "Number of HTTP requests currently being served.",
		},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "pulse",
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "HTTP response size distribution in bytes.",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
		},
		[]string{"method", "path"},
	)

	wsActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "pulse",
			Subsystem: "ws",
			Name:      "active_connections",
			Help:      "Number of active WebSocket connections.",
		},
	)
)

// Metrics returns Gin middleware that records Prometheus metrics for every request.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use the route pattern (e.g. "/api/v1/users/:id") rather than the
		// actual path to keep cardinality bounded.
		start := time.Now()

		httpActiveConnections.Inc()
		defer httpActiveConnections.Dec()

		c.Next()

		// FullPath returns the registered route pattern, or empty for 404s.
		routePattern := c.FullPath()
		if routePattern == "" {
			routePattern = "unmatched"
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		elapsed := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(status, method, routePattern).Inc()
		httpRequestDuration.WithLabelValues(method, routePattern).Observe(elapsed)
		httpResponseSize.WithLabelValues(method, routePattern).Observe(float64(c.Writer.Size()))
	}
}

// WSConnectionOpened increments the active WebSocket connections gauge.
func WSConnectionOpened() {
	wsActiveConnections.Inc()
}

// WSConnectionClosed decrements the active WebSocket connections gauge.
func WSConnectionClosed() {
	wsActiveConnections.Dec()
}
