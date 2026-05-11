package mount

import (
	"net/http"
	"strconv"

	"xray2wg/backend/internal/api/apideps"
	sqldb "xray2wg/backend/internal/infrastructure/db"

	"github.com/labstack/echo/v4"
)

func MountAudit(api *echo.Group, d *apideps.Deps) {
	repo := d.AuditDB

	api.GET("/audit", func(c echo.Context) error {
		level := c.QueryParam("level")
		search := c.QueryParam("search")
		limit, _ := strconv.Atoi(c.QueryParam("limit"))
		offset, _ := strconv.Atoi(c.QueryParam("offset"))

		page, err := repo.List(sqldb.AuditLogFilter{
			Level:  level,
			Search: search,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]any{
			"items": page.Items,
			"total": page.Total,
		})
	})
}
