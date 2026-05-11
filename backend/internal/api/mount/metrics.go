package mount

import (
	"strings"

	"xray2wg/backend/internal/api/middleware"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsRegistry configures Prometheus HTTP exposition (production hardening contract).
type MetricsRegistry struct {
	APIKey string
}

// NewMetricsRegistry constructs a registry. Pass the raw METRICS_API_KEY value (may be empty).
func NewMetricsRegistry(apiKey string) *MetricsRegistry {
	return &MetricsRegistry{APIKey: apiKey}
}

// Mount registers GET /metrics (Prometheus). When APIKey is set, clients must send header
// X-Metrics-Key. When empty, only loopback addresses may scrape.
func (m *MetricsRegistry) Mount(e *echo.Echo) {
	key := strings.TrimSpace(m.APIKey)
	// MetricsAccess is passed as route-level middleware (not group middleware) to avoid
	// Echo's Group.Use() side-effect: it auto-registers RouteNotFound("/*") with the same
	// middleware, which hijacks catch-all routing and breaks the static SPA fallback.
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()), middleware.MetricsAccess(key))
}

// MountMetrics registers GET /metrics using a default MetricsRegistry (backward compatible).
func MountMetrics(e *echo.Echo, apiKey string) {
	NewMetricsRegistry(apiKey).Mount(e)
}
