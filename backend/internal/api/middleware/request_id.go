package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const echoRequestIDKey = "request_id"

// RequestID ensures X-Request-ID on the response and attaches a per-request zerolog logger to the request context.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rid := c.Request().Header.Get(echo.HeaderXRequestID)
			if rid == "" {
				rid = genRequestID()
			}
			c.Request().Header.Set(echo.HeaderXRequestID, rid)
			c.Response().Header().Set(echo.HeaderXRequestID, rid)
			c.Set(echoRequestIDKey, rid)

			lg := log.With().Str("request_id", rid).Logger()
			req := c.Request().WithContext(lg.WithContext(c.Request().Context()))
			c.SetRequest(req)
			return next(c)
		}
	}
}

func genRequestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "req-fallback"
	}
	return hex.EncodeToString(b)
}

// GetRequestID returns the request id from Echo context (empty if middleware did not run).
func GetRequestID(c echo.Context) string {
	v, ok := c.Get(echoRequestIDKey).(string)
	if !ok {
		return ""
	}
	return v
}

// GetLogger returns the request-scoped logger from context (may be a disabled logger).
func GetLogger(c echo.Context) *zerolog.Logger {
	return zerolog.Ctx(c.Request().Context())
}
