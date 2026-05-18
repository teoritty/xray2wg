package mount

import (
	"net/http"

	"xray2wg/backend/internal/api/apideps"
	"xray2wg/backend/internal/app"
	"xray2wg/backend/internal/domain"

	"github.com/labstack/echo/v4"
)

// MountSubscriptions registers /api/v1/subscriptions* (JWT group).
func MountSubscriptions(api *echo.Group, d *apideps.Deps) {
	sapi := app.NewSubscriptionsAPI(d.SubRepo, d.Subs, d.NodeHealth)

	api.GET("/subscriptions", func(c echo.Context) error {
		list, err := sapi.List(c.Request().Context())
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, list)
	})

	api.POST("/subscriptions", func(c echo.Context) error {
		var in struct {
			Name            string `json:"name"`
			URL             string `json:"url"`
			RefreshInterval int64  `json:"refresh_interval"`
		}
		if err := c.Bind(&in); err != nil {
			return err
		}
		su, err := sapi.Add(c.Request().Context(), app.SubscriptionCreateInput{
			Name: in.Name, URL: in.URL, RefreshInterval: in.RefreshInterval,
		})
		if err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, su)
	})

	// Manual-node creation accepts two body shapes:
	//   1. {"vless_uri": "vless://..."}             — fast path: paste a share-link.
	//   2. {"uuid": "...", "network": "xhttp", ...} — structured: fields from a UI form.
	// The transport / security parameters in the structured form live inside opaque JSON
	// payloads so this endpoint never has to grow when a new transport is registered.
	api.POST("/subscriptions/manual-nodes", func(c echo.Context) error {
		var in struct {
			VlessURI string             `json:"vless_uri"`
			Manual   *app.ManualNodeInput `json:"manual,omitempty"`
		}
		if err := c.Bind(&in); err != nil {
			return domain.ErrValidation
		}
		var (
			node *domain.VlessNode
			err  error
		)
		switch {
		case in.Manual != nil:
			node, err = sapi.AddManualNodeStructured(c.Request().Context(), *in.Manual)
		default:
			node, err = sapi.AddManualVlessNode(c.Request().Context(), in.VlessURI)
		}
		if err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, node)
	})

	api.PUT("/subscriptions/manual-nodes/:nodeId", func(c echo.Context) error {
		nodeID, err := app.ParseID64(c.Param("nodeId"))
		if err != nil || nodeID <= 0 {
			return domain.ErrValidation
		}
		var in struct {
			VlessURI string             `json:"vless_uri"`
			Manual   *app.ManualNodeInput `json:"manual,omitempty"`
		}
		if err := c.Bind(&in); err != nil {
			return domain.ErrValidation
		}
		var node *domain.VlessNode
		switch {
		case in.Manual != nil:
			node, err = sapi.UpdateManualNodeStructured(c.Request().Context(), nodeID, *in.Manual)
		default:
			node, err = sapi.UpdateManualVlessNode(c.Request().Context(), nodeID, in.VlessURI)
		}
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, node)
	})

	api.DELETE("/subscriptions/manual-nodes/:nodeId", func(c echo.Context) error {
		nodeID, err := app.ParseID64(c.Param("nodeId"))
		if err != nil || nodeID <= 0 {
			return domain.ErrValidation
		}
		if err := sapi.DeleteManualVlessNode(c.Request().Context(), nodeID); err != nil {
			return err
		}
		return c.NoContent(http.StatusNoContent)
	})

	api.GET("/subscriptions/:id", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		su, err := sapi.Get(c.Request().Context(), id)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, su)
	})

	api.PUT("/subscriptions/:id", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		var su domain.Subscription
		if err := c.Bind(&su); err != nil {
			return err
		}
		su.ID = id
		return sapi.Update(c.Request().Context(), &su)
	})

	api.DELETE("/subscriptions/:id", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		return sapi.Delete(c.Request().Context(), id)
	})

	api.POST("/subscriptions/:id/refresh", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		return sapi.Refresh(c.Request().Context(), id)
	})

	api.GET("/subscriptions/:id/nodes", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		nodes, err := sapi.ListNodes(c.Request().Context(), id)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, nodes)
	})
}
