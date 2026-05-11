package middleware

import (
	"net/http"

	"xray2wg/backend/internal/telemetry"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// TokenParser parses and validates JWT access tokens (for WebSocket upgrade requests).
type TokenParser interface {
	ParseToken(token string) (jwt.MapClaims, error)
}

// WebSocketAuth validates the access token from the HttpOnly access_token cookie only.
func WebSocketAuth(p TokenParser) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ck, err := c.Cookie("access_token")
			if err != nil || ck.Value == "" {
				if telemetry.AuthFailuresTotal != nil {
					telemetry.AuthFailuresTotal.Inc()
				}
				return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
			}
			cl, err := p.ParseToken(ck.Value)
			if err != nil {
				if telemetry.AuthFailuresTotal != nil {
					telemetry.AuthFailuresTotal.Inc()
				}
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			if typ, ok := cl["typ"].(string); ok && typ != "access" {
				if telemetry.AuthFailuresTotal != nil {
					telemetry.AuthFailuresTotal.Inc()
				}
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			c.Set("claims", cl)
			return next(c)
		}
	}
}
