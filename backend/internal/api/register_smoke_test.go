package api

import (
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func routePaths(e *echo.Echo) []string {
	var out []string
	for _, r := range e.Routes() {
		if r.Path != "" {
			out = append(out, r.Path)
		}
	}
	return out
}

func TestRegisterSmokeRoutePaths(t *testing.T) {
	e, _, cancel := newAuthTestEcho(t)
	defer cancel()
	paths := strings.Join(routePaths(e), " ")
	for _, want := range []string{
		"/api/v1/auth/login",
		"/api/v1/auth/me",
		"/api/v1/auth/setup-status",
		"/api/v1/tunnels",
		"/api/v1/ws/stats",
		"/api/v1/health",
		"/api/v1/ready",
	} {
		if !strings.Contains(paths, want) {
			t.Fatalf("missing route %q in %s", want, paths)
		}
	}
}
