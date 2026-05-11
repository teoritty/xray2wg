package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCSPHeader(t *testing.T) {
	t.Setenv("CSP_OFF", "")
	e := echo.New()
	e.Use(CSP())
	e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
	h := rec.Header().Get("Content-Security-Policy")
	if h == "" || !strings.Contains(h, "default-src") {
		t.Fatalf("csp: %q", h)
	}
	if strings.Contains(h, "'unsafe-inline'") || strings.Contains(h, "'unsafe-eval'") {
		t.Fatalf("unsafe directive in CSP: %q", h)
	}
}

func TestCSPOff(t *testing.T) {
	t.Setenv("CSP_OFF", "true")
	e := echo.New()
	e.Use(CSP())
	e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Header().Get("Content-Security-Policy") != "" {
		t.Fatal("expected no CSP when CSP_OFF=true")
	}
}
