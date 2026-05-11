package mount

import (
	"context"
	"database/sql"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
)

// Health exposes liveness and readiness probes.
type Health struct {
	DB        *sql.DB
	startedAt time.Time
	ready     atomic.Bool
}

// NewHealth constructs a health handler; db may be nil (readiness will fail until DB is set).
func NewHealth(db *sql.DB) *Health {
	return &Health{DB: db, startedAt: time.Now().UTC()}
}

// MarkReady marks the process ready to serve traffic (call after full initialization).
func (h *Health) MarkReady() {
	if h == nil {
		return
	}
	h.ready.Store(true)
}

// Liveness always succeeds if the process is running.
func (h *Health) Liveness(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{"status": "alive"})
}

// Readiness reports 503 until MarkReady and DB ping succeed.
func (h *Health) Readiness(c echo.Context) error {
	if h == nil || !h.ready.Load() {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"status": "not_ready"})
	}
	if h.DB == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"status": "no_db"})
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
	defer cancel()
	if err := h.DB.PingContext(ctx); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{
			"status": "unready",
			"reason": "database unreachable",
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "ready"})
}
