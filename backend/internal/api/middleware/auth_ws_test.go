package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type mockParser struct {
	mc jwt.MapClaims
	err error
}

func (m mockParser) ParseToken(token string) (jwt.MapClaims, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.mc, nil
}

func TestWebSocketAuthValidCookie(t *testing.T) {
	e := echo.New()
	e.GET("/ws", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, WebSocketAuth(mockParser{mc: jwt.MapClaims{"typ": "access", "user_id": float64(1)}}))

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "valid.jwt.here"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestWebSocketAuthMissingCookie(t *testing.T) {
	e := echo.New()
	e.GET("/ws", func(c echo.Context) error { return c.NoContent(http.StatusOK) }, WebSocketAuth(mockParser{}))

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestWebSocketAuthInvalidToken(t *testing.T) {
	e := echo.New()
	e.GET("/ws", func(c echo.Context) error { return c.NoContent(http.StatusOK) },
		WebSocketAuth(mockParser{err: errors.New("invalid")}))

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "bad"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestWebSocketAuthQueryTokenIgnored(t *testing.T) {
	e := echo.New()
	e.GET("/ws", func(c echo.Context) error { return c.NoContent(http.StatusOK) },
		WebSocketAuth(mockParser{mc: jwt.MapClaims{"typ": "access"}}))

	req := httptest.NewRequest(http.MethodGet, "/ws?token=should-be-ignored&ignored=1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without cookie (query string must not authenticate), got %d", rec.Code)
	}
}
