package mount

import (
	"net/http"

	"xray2wg/backend/internal/api/apideps"
	"xray2wg/backend/internal/api/middleware"
	"xray2wg/backend/internal/app"
	"xray2wg/backend/internal/service"

	"github.com/labstack/echo/v4"
)

// MountSession registers authenticated session introspection routes (JWT group).
func MountSession(api *echo.Group, _ *apideps.Deps) {
	sapi := app.NewSessionAPI()

	api.GET("/auth/me", func(c echo.Context) error {
		mc := middleware.BearerClaims(c)
		ac, ok := service.ClaimsFromAccess(mc)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		return c.JSON(http.StatusOK, sapi.MePayload(ac.UserID))
	})
}
