package mount

import (
	"errors"
	"net/http"

	"xray2wg/backend/internal/api/apideps"
	"xray2wg/backend/internal/app"
	"xray2wg/backend/internal/domain"

	"github.com/labstack/echo/v4"
)

// MountPublicAuth registers unauthenticated /api/v1/auth/* and GET /api/v1/auth/setup-status (bootstrap probe).
func MountPublicAuth(auth *echo.Group, d *apideps.Deps) {
	aapi := app.NewPublicAuthAPI(d.Set, d.Auth)

	auth.POST("/auth/login", func(c echo.Context) error {
		var in struct {
			Password string `json:"password"`
		}
		if err := c.Bind(&in); err != nil {
			return domain.ErrValidation
		}
		access, refresh, err := aapi.Login(c.Request().Context(), in.Password, c.RealIP())
		if err != nil {
			return err
		}
		setTokenCookies(c, access, refresh)
		return c.NoContent(http.StatusNoContent)
	})

	auth.POST("/auth/bootstrap", func(c echo.Context) error {
		var in struct {
			Password string `json:"password"`
			Confirm  string `json:"confirm"`
		}
		if err := c.Bind(&in); err != nil {
			return domain.ErrValidation
		}
		access, refresh, err := aapi.Bootstrap(c.Request().Context(), app.BootstrapInput{
			Password: in.Password,
			Confirm:  in.Confirm,
		})
		if err != nil {
			return err
		}
		setTokenCookies(c, access, refresh)
		return c.NoContent(http.StatusNoContent)
	})

	auth.GET("/auth/setup-status", func(c echo.Context) error {
		body, err := aapi.SetupStatusJSON(c.Request().Context(), c.Scheme())
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, body)
	})

	auth.POST("/auth/refresh", func(c echo.Context) error {
		ck, err := c.Cookie("refresh_token")
		if err != nil || ck.Value == "" {
			return domain.ErrUnauthorized
		}
		access, refresh, err := aapi.RotateRefresh(ck.Value)
		if err != nil {
			if errors.Is(err, domain.ErrUnauthorized) {
				return domain.ErrUnauthorized
			}
			return err
		}
		setTokenCookies(c, access, refresh)
		return c.NoContent(http.StatusNoContent)
	})

	auth.POST("/auth/logout", func(c echo.Context) error {
		var rt string
		if ck, err := c.Cookie("refresh_token"); err == nil {
			rt = ck.Value
		}
		aapi.RevokeRefresh(rt)
		clearTokenCookies(c)
		return c.NoContent(http.StatusNoContent)
	})
}
