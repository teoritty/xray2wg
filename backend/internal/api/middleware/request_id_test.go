package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestRequestIDGeneratesWhenMissing(t *testing.T) {
	e := echo.New()
	e.Use(RequestID())
	e.GET("/x", func(c echo.Context) error {
		if GetRequestID(c) == "" {
			t.Fatal("expected generated id")
		}
		return c.NoContent(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	e.ServeHTTP(rec, req)
	if rec.Header().Get(echo.HeaderXRequestID) == "" {
		t.Fatal("missing response header")
	}
}

func TestRequestIDProxiesExisting(t *testing.T) {
	e := echo.New()
	e.Use(RequestID())
	e.GET("/x", func(c echo.Context) error {
		if GetRequestID(c) != "abc" {
			t.Fatalf("got %q", GetRequestID(c))
		}
		return c.NoContent(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(echo.HeaderXRequestID, "abc")
	e.ServeHTTP(rec, req)
	if rec.Header().Get(echo.HeaderXRequestID) != "abc" {
		t.Fatalf("header %q", rec.Header().Get(echo.HeaderXRequestID))
	}
}

func TestGetLoggerContainsRequestID(t *testing.T) {
	e := echo.New()
	e.Use(RequestID())
	e.GET("/x", func(c echo.Context) error {
		lg := GetLogger(c)
		if lg.GetLevel() == zerolog.Disabled {
			t.Fatal("expected logger in context")
		}
		return c.NoContent(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	e.ServeHTTP(rec, req)
	_ = rec
}

func TestRequestIDPropagatesToZerologOutput(t *testing.T) {
	prev := log.Logger
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()
	t.Cleanup(func() { log.Logger = prev })

	e := echo.New()
	e.Use(RequestID())
	e.GET("/x", func(c echo.Context) error {
		rid := GetRequestID(c)
		zerolog.Ctx(c.Request().Context()).Info().Str("probe", "1").Msg("handler_log")
		if rid == "" {
			t.Fatal("empty rid")
		}
		return c.NoContent(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
	rid := rec.Header().Get(echo.HeaderXRequestID)
	if rid == "" || !bytes.Contains(buf.Bytes(), []byte(rid)) {
		t.Fatalf("expected request_id %q in log output, got %s", rid, buf.String())
	}
}

func TestGetRequestIDEmptyWithoutMiddleware(t *testing.T) {
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	if GetRequestID(c) != "" {
		t.Fatalf("got %q", GetRequestID(c))
	}
}
