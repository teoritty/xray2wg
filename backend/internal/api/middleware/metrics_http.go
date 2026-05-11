package middleware

import (
	"net/http"
	"strconv"
	"time"

	"xray2wg/backend/internal/telemetry"

	"github.com/labstack/echo/v4"
)

// MetricsHTTP records Prometheus counters and histograms for each HTTP request.
func MetricsHTTP() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			status := c.Response().Status
			if status == 0 {
				if err != nil {
					if he, ok := err.(*echo.HTTPError); ok {
						status = he.Code
					} else {
						status = http.StatusInternalServerError
					}
				} else {
					status = http.StatusOK
				}
			}
			path := c.Path()
			if path == "" {
				path = "unknown"
			}
			code := strconv.Itoa(status)
			telemetry.HTTPRequestsTotal.WithLabelValues(c.Request().Method, path, code).Inc()
			telemetry.HTTPDuration.WithLabelValues(c.Request().Method, path, code).Observe(time.Since(start).Seconds())
			return err
		}
	}
}
