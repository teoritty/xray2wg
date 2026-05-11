package mount

import (
	"net/http"

	"xray2wg/backend/internal/api/apideps"
	"xray2wg/backend/internal/app"

	"github.com/labstack/echo/v4"
)

// MountPeers registers /api/v1/peers (JWT group).
func MountPeers(api *echo.Group, d *apideps.Deps) {
	papi := app.NewPeersAPI(d.PeerRepo)

	api.GET("/peers", func(c echo.Context) error {
		list, err := papi.ListAllWithTunnel(c.Request().Context())
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, list)
	})
}
