package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/telemetry"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type errPayload struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}

func TestErrorHandlerSQLMasked(t *testing.T) {
	var buf bytes.Buffer
	lg := zerolog.New(&buf).With().Timestamp().Logger()
	ctx := lg.WithContext(context.Background())

	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	sqlErr := errors.New("dial tcp 127.0.0.1:5432: connection refused")
	ErrorHandler(sqlErr, c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	var body errPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "INTERNAL" || body.Error.Message != "internal server error" {
		t.Fatalf("response: %+v", body.Error)
	}
	raw := rec.Body.String()
	if strings.Contains(raw, "5432") || strings.Contains(raw, "dial tcp") {
		t.Fatalf("response leaks sql details: %s", raw)
	}
	if !strings.Contains(buf.String(), "dial tcp") {
		t.Fatalf("expected full error in logs, got: %s", buf.String())
	}
}

func TestErrorHandlerFilePathMasked(t *testing.T) {
	var buf bytes.Buffer
	lg := zerolog.New(&buf).With().Timestamp().Logger()
	ctx := lg.WithContext(context.Background())

	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	pathErr := fmt.Errorf("read config: %w", &os.PathError{Op: "open", Path: "/secret/app.db", Err: errors.New("denied")})
	ErrorHandler(pathErr, c)

	var body errPayload
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Message != "internal server error" {
		t.Fatalf("got %q", body.Error.Message)
	}
	raw := rec.Body.String()
	if strings.Contains(raw, "/secret/") || strings.Contains(raw, "app.db") {
		t.Fatalf("path leaked: %s", raw)
	}
	if !strings.Contains(buf.String(), "/secret/app.db") {
		t.Fatalf("expected path in logs: %s", buf.String())
	}
}

func TestErrorHandlerDomainNotFound(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	ErrorHandler(domain.NewNotFoundError("tunnel 7 is gone"), c)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
	var body errPayload
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Code != "NOT_FOUND" || body.Error.Message != "tunnel 7 is gone" {
		t.Fatalf("got %+v", body.Error)
	}
}

func TestErrorHandlerDomainUnauthorized(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	ErrorHandler(domain.NewUnauthorizedError("bad credentials"), c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
	var body errPayload
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Code != "UNAUTHORIZED" || body.Error.Message != "bad credentials" {
		t.Fatalf("got %+v", body.Error)
	}
}

func TestErrorHandlerEcho5xxMasked(t *testing.T) {
	var buf bytes.Buffer
	lg := zerolog.New(&buf).With().Timestamp().Logger()
	ctx := lg.WithContext(context.Background())

	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	ErrorHandler(echo.NewHTTPError(http.StatusBadGateway, "upstream dsn postgres://secret"), c)

	var body errPayload
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Message != "internal server error" {
		t.Fatalf("got %q", body.Error.Message)
	}
	if strings.Contains(rec.Body.String(), "postgres") {
		t.Fatal("upstream detail leaked")
	}
}

func TestErrorHandlerRequestIDInResponse(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	e.Use(RequestID())
	e.GET("/x", func(c echo.Context) error {
		return domain.NewNotFoundError("missing resource")
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var body errPayload
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.RequestID == "" {
		t.Fatal("expected request_id in error payload")
	}
	if got := rec.Header().Get(echo.HeaderXRequestID); got != "" && body.Error.RequestID != got {
		t.Fatalf("request_id mismatch body=%q header=%q", body.Error.RequestID, got)
	}
}

func TestErrorHandlerUsesZerologTestWriter(t *testing.T) {
	tw := zerolog.TestWriter{T: t}
	lg := zerolog.New(tw).Level(zerolog.ErrorLevel)
	ctx := lg.WithContext(context.Background())

	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	ErrorHandler(errors.New("boom-internal-detail"), c)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	// TestWriter forwards logs to t.Logf; successful execution implies logging path ran without panic.
	_ = rec.Body.String()
}

func TestErrorHandlerSentinelUnauthorized(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	ErrorHandler(domain.ErrUnauthorized, c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestErrorHandlerDomainConflict(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ErrorHandler(domain.NewConflictError("already exists"), c)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d", rec.Code)
	}
	var body errPayload
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Code != "CONFLICT" {
		t.Fatalf("code %q", body.Error.Code)
	}
}

func TestErrorHandlerDomainForbidden(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ErrorHandler(domain.NewForbiddenError("no access"), c)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d", rec.Code)
	}
	var body errPayload
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Code != "FORBIDDEN" {
		t.Fatalf("code %q", body.Error.Code)
	}
}

func TestErrorHandlerWrappedGORMRecordNotFoundMasked(t *testing.T) {
	var buf bytes.Buffer
	lg := zerolog.New(&buf).With().Timestamp().Logger()
	ctx := lg.WithContext(context.Background())

	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	wrapped := fmt.Errorf("query failed: %w", gorm.ErrRecordNotFound)
	ErrorHandler(wrapped, c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	raw := rec.Body.String()
	if strings.Contains(strings.ToLower(raw), "record not found") {
		t.Fatalf("response leaks gorm detail: %s", raw)
	}
	if !strings.Contains(strings.ToLower(buf.String()), "record not found") {
		t.Fatalf("expected gorm detail in logs: %s", buf.String())
	}
}

func TestErrorHandlerInternalPasswordInLogsNotResponse(t *testing.T) {
	var buf bytes.Buffer
	lg := zerolog.New(&buf).With().Timestamp().Logger()
	ctx := lg.WithContext(context.Background())

	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	secret := "VerySecretDBPassword_9f3a"
	ErrorHandler(fmt.Errorf("connection refused: password=%s dial tcp", secret), c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), secret) {
		t.Fatalf("password leaked to client: %s", rec.Body.String())
	}
	if !strings.Contains(buf.String(), secret) {
		t.Fatalf("expected password in logs: %s", buf.String())
	}
}

func TestMain(m *testing.M) {
	telemetry.Register(prometheus.NewRegistry())
	// Global zerolog used when context has no logger (loggerFor fallback).
	log.Logger = zerolog.New(io.Discard)
	os.Exit(m.Run())
}
