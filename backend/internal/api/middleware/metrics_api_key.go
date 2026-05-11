package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const metricsHeaderKey = "X-Metrics-Key"

// MetricsAccess protects Prometheus scraping: shared secret when apiKey is non-empty,
// otherwise only loopback clients (127.0.0.1 / ::1) may access /metrics.
func MetricsAccess(apiKey string) echo.MiddlewareFunc {
	key := strings.TrimSpace(apiKey)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if key != "" {
				if c.Request().Header.Get(metricsHeaderKey) != key {
					return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
				}
				return next(c)
			}
			if isMetricsLoopback(c.Request()) {
				return next(c)
			}
			return echo.NewHTTPError(http.StatusForbidden, "forbidden")
		}
	}
}

func isMetricsLoopback(r *http.Request) bool {
	addr := strings.TrimSpace(r.RemoteAddr)
	if addr == "" {
		return false
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		ip := net.ParseIP(addr)
		return ip != nil && ip.IsLoopback()
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
