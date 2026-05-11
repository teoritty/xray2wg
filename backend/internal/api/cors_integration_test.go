package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"xray2wg/backend/internal/security/origin"

	"github.com/labstack/echo/v4"
	echo_mid "github.com/labstack/echo/v4/middleware"
)

// corsEcho wires the same CORS policy as Register (global middleware + simple route).
func corsEcho(t *testing.T, allowedOrigins string) *echo.Echo {
	t.Helper()
	oc, err := origin.NewConfig(allowedOrigins)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.Use(echo_mid.CORSWithConfig(echo_mid.CORSConfig{
		AllowOriginFunc: func(origin string) (bool, error) {
			return oc.AllowOrigin(origin), nil
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Cookie"},
		AllowCredentials: true,
	}))
	e.GET("/ping", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	return e
}

func TestCORSPreflightAllowedOrigin(t *testing.T) {
	e := corsEcho(t, "https://localhost:5173")

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set(echo.HeaderOrigin, "https://localhost:5173")
	req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: %d", rec.Code)
	}
	if got := rec.Header().Get(echo.HeaderAccessControlAllowOrigin); got != "https://localhost:5173" {
		t.Fatalf("ACAO: got %q want https://localhost:5173", got)
	}
}

func TestCORSPreflightDisallowedOrigin(t *testing.T) {
	e := corsEcho(t, "https://localhost:5173")

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set(echo.HeaderOrigin, "https://evil.example")
	req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Header().Get(echo.HeaderAccessControlAllowOrigin) != "" {
		t.Fatalf("disallowed origin must not get ACAO, got %q", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
	}
}
