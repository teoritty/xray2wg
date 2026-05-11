package middleware

import (
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

// CSP sets a Content-Security-Policy suitable for the SPA (Vite bundle + Google Fonts).
// Set CSP_OFF=true to disable (local tooling only).
func CSP() echo.MiddlewareFunc {
	policy := strings.Join([]string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com data:",
		"connect-src 'self' ws: wss:",
		"img-src 'self' data: blob:",
		"frame-ancestors 'none'",
		"base-uri 'none'",
		"form-action 'self'",
	}, "; ")
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if strings.EqualFold(strings.TrimSpace(os.Getenv("CSP_OFF")), "true") {
				return next(c)
			}
			c.Response().Header().Set("Content-Security-Policy", policy)
			return next(c)
		}
	}
}
