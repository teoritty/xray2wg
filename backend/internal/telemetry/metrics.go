package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics (register once via Register).
var (
	HTTPRequestsTotal *prometheus.CounterVec
	HTTPDuration      *prometheus.HistogramVec
	ActiveTunnels     prometheus.Gauge
	ActiveWSConns     prometheus.Gauge
	AuthFailuresTotal prometheus.Counter
)

// Register wires application metrics into the default Prometheus registry.
func Register(reg prometheus.Registerer) {
	factory := promauto.With(reg)
	HTTPRequestsTotal = factory.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "code"})
	HTTPDuration = factory.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "code"})
	ActiveTunnels = factory.NewGauge(prometheus.GaugeOpts{
		Name: "active_tunnels",
		Help: "Tunnels running in process",
	})
	ActiveWSConns = factory.NewGauge(prometheus.GaugeOpts{
		Name: "active_websocket_connections",
		Help: "Connected WebSocket clients",
	})
	AuthFailuresTotal = factory.NewCounter(prometheus.CounterOpts{
		Name: "auth_failures_total",
		Help: "Failed authentication attempts",
	})
}
