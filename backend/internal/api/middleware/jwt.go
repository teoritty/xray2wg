package middleware

import (
	"net/http"

	"xray2wg/backend/internal/service"
	"xray2wg/backend/internal/telemetry"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func JWT(auth *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var tok string
			if ck, err := c.Cookie("access_token"); err == nil {
				tok = ck.Value
			}
			if tok == "" {
				if telemetry.AuthFailuresTotal != nil {
					telemetry.AuthFailuresTotal.Inc()
				}
				return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
			}
			cl, err := auth.ParseToken(tok)
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
				return echo.NewHTTPError(http.StatusUnauthorized, "wrong token")
			}
			c.Set("claims", cl)
			return next(c)
		}
	}
}

func BearerClaims(c echo.Context) jwt.MapClaims {
	v, ok := c.Get("claims").(jwt.MapClaims)
	if !ok {
		return nil
	}
	return v
}
