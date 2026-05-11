package mount

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"xray2wg/backend/internal/api/middleware"
	"xray2wg/backend/internal/telemetry"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestMetricsEndpointPrometheusFormat(t *testing.T) {
	if telemetry.HTTPRequestsTotal != nil {
		telemetry.HTTPRequestsTotal.WithLabelValues("GET", "/x", "200").Inc()
	}
	if telemetry.HTTPDuration != nil {
		telemetry.HTTPDuration.WithLabelValues("GET", "/x", "200").Observe(0.001)
	}
	if telemetry.ActiveTunnels != nil {
		telemetry.ActiveTunnels.Set(1)
	}
	if telemetry.ActiveWSConns != nil {
		telemetry.ActiveWSConns.Set(1)
	}
	if telemetry.AuthFailuresTotal != nil {
		telemetry.AuthFailuresTotal.Inc()
	}

	e := echo.New()
	key := "secret-metrics-key"
	e.GET("/metrics", echo.WrapHandler(promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})), middleware.MetricsAccess(key))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("X-Metrics-Key", key)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if body == "" {
		t.Fatal("empty metrics body")
	}
	for _, name := range []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"active_tunnels",
		"active_websocket_connections",
		"auth_failures_total",
	} {
		if !strings.Contains(body, name) {
			t.Fatalf("metrics body missing %q", name)
		}
	}
}

func TestMetricsLoopbackWithoutKey(t *testing.T) {
	e := echo.New()
	e.GET("/metrics", echo.WrapHandler(promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})), middleware.MetricsAccess(""))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestMetricsForbiddenNonLoopbackWithoutKey(t *testing.T) {
	e := echo.New()
	e.GET("/metrics", echo.WrapHandler(promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})), middleware.MetricsAccess(""))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "198.51.100.2:1234"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestMetricsActiveTunnelsGaugeRegistered(t *testing.T) {
	telemetry.ActiveTunnels.Set(3)
}
