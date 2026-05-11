package mount

import (
	"net/http"

	"xray2wg/backend/internal/api/apideps"
	"xray2wg/backend/internal/app"
	"xray2wg/backend/internal/domain"

	"github.com/labstack/echo/v4"
)

// MountSettings registers /api/v1/settings* (JWT group).
func MountSettings(api *echo.Group, d *apideps.Deps) {
	sapi := app.NewSettingsAPI(d.Set, d.Auth, d.Subs)

	api.GET("/settings", func(c echo.Context) error {
		sh, err := sapi.GetServerHost(c.Request().Context())
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]string{"server_host": sh})
	})

	api.PUT("/settings", func(c echo.Context) error {
		var in struct {
			ServerHost string `json:"server_host"`
		}
		if err := c.Bind(&in); err != nil {
			return err
		}
		return sapi.SetServerHost(c.Request().Context(), in.ServerHost)
	})

	api.PUT("/settings/password", func(c echo.Context) error {
		var in struct {
			Old string `json:"old_password"`
			New string `json:"new_password"`
		}
		if err := c.Bind(&in); err != nil {
			return domain.ErrValidation
		}
		return sapi.ChangeAdminPassword(c.Request().Context(), app.PasswordChangeInput{Old: in.Old, New: in.New})
	})

	api.GET("/settings/export", func(c echo.Context) error {
		b, err := sapi.ExportMinimal(c.Request().Context())
		if err != nil {
			return err
		}
		return c.Blob(http.StatusOK, "application/json", b)
	})

	api.POST("/settings/import", func(c echo.Context) error {
		b, err := app.ReadLimitedBody(c.Request().Body, 1<<20)
		if err != nil {
			return err
		}
		return sapi.ImportMinimal(c.Request().Context(), b)
	})
}
