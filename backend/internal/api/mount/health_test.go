package mount

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
)

func TestLivenessAlways200(t *testing.T) {
	e := echo.New()
	h := NewHealth(nil)
	g := e.Group("/api/v1")
	g.GET("/health", h.Liveness)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "alive") {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestReadiness503BeforeMarkReady(t *testing.T) {
	e := echo.New()
	h := NewHealth(nil)
	g := e.Group("/api/v1")
	g.GET("/ready", h.Readiness)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestReadinessDBUnreachable(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "ready.db"))
	if err != nil {
		t.Fatal(err)
	}
	h := NewHealth(db)
	h.MarkReady()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	g := e.Group("/api/v1")
	g.GET("/ready", h.Readiness)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code %d body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "database unreachable") {
		t.Fatalf("expected database unreachable reason, got %s", rec.Body.String())
	}
}
