package mount

import (
	"net/http"

	"xray2wg/backend/internal/api/apideps"
	"xray2wg/backend/internal/app"
	"xray2wg/backend/internal/domain"

	"github.com/labstack/echo/v4"
)

// MountTunnels registers /api/v1/tunnels*, /api/v1/stats/summary, and tunnel-scoped peer routes (JWT group).
func MountTunnels(api *echo.Group, d *apideps.Deps) {
	tapi := app.NewTunnelsAPI(
		d.TunRepo,
		d.SubRepo,
		d.PeerRepo,
		d.Set,
		d.Tunnels,
		d.Peers,
		d.Stats,
		d.EventLog,
		d.MasterKey,
		d.ManualSubID,
	)

	api.GET("/tunnels", func(c echo.Context) error {
		list, err := tapi.ListTunnels(c.Request().Context())
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, list)
	})

	api.POST("/tunnels", func(c echo.Context) error {
		var in struct {
			Name              string  `json:"name"`
			ListenPort        int     `json:"listen_port"`
			WgAddress         string  `json:"wg_address"`
			DNS               string  `json:"dns"`
			MTU               int     `json:"mtu"`
			SubscriptionID    *int64  `json:"subscription_id"`
			ActiveNodeID      *int64  `json:"active_node_id"`
			NodeIDs           []int64 `json:"node_ids"`
			BalancingStrategy string  `json:"balancing_strategy"`
			VlessURI          string  `json:"vless_uri"`
		}
		if err := c.Bind(&in); err != nil {
			return err
		}
		iface, err := tapi.CreateTunnel(c.Request().Context(), app.CreateTunnelInput{
			Name:              in.Name,
			ListenPort:        in.ListenPort,
			WgAddress:         in.WgAddress,
			DNS:               in.DNS,
			MTU:               in.MTU,
			SubscriptionID:    in.SubscriptionID,
			ActiveNodeID:      in.ActiveNodeID,
			NodeIDs:           in.NodeIDs,
			BalancingStrategy: in.BalancingStrategy,
			VlessURI:          in.VlessURI,
		})
		if err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, iface)
	})

	api.GET("/stats/summary", func(c echo.Context) error {
		body, err := tapi.StatsSummary(c.Request().Context())
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, body)
	})

	api.GET("/tunnels/:id", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		iface, err := tapi.GetTunnel(c.Request().Context(), id)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, iface)
	})

	api.PUT("/tunnels/:id", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		var iface domain.WgInterface
		if err := c.Bind(&iface); err != nil {
			return err
		}
		iface.ID = id
		return tapi.UpdateTunnel(c.Request().Context(), id, &iface)
	})

	api.DELETE("/tunnels/:id", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		return tapi.DeleteTunnel(c.Request().Context(), id)
	})

	api.POST("/tunnels/:id/start", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		return tapi.StartTunnel(c.Request().Context(), id)
	})

	api.POST("/tunnels/:id/stop", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		return tapi.StopTunnel(c.Request().Context(), id)
	})

	api.GET("/tunnels/:id/nodes", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		result, err := tapi.GetTunnelNodes(c.Request().Context(), id)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, result)
	})

	api.PUT("/tunnels/:id/nodes", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		var in struct {
			NodeIDs  []int64 `json:"node_ids"`
			Strategy string  `json:"strategy"`
		}
		if err := c.Bind(&in); err != nil {
			return err
		}
		if err := tapi.SetTunnelNodes(c.Request().Context(), id, in.NodeIDs, in.Strategy); err != nil {
			return err
		}
		return c.NoContent(http.StatusNoContent)
	})

	api.GET("/tunnels/:id/stats", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		win := c.QueryParam("window")
		rows, err := tapi.TunnelStats(c.Request().Context(), id, win)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, rows)
	})

	api.GET("/tunnels/:id/peers", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		list, err := tapi.ListTunnelPeers(c.Request().Context(), id)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, list)
	})

	api.POST("/tunnels/:id/peers", func(c echo.Context) error {
		id, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		var in struct {
			Name      string `json:"name"`
			PublicKey string `json:"public_key"`
			ClientIP  string `json:"client_address"`
		}
		if err := c.Bind(&in); err != nil {
			return err
		}
		p, err := tapi.CreateTunnelPeer(c.Request().Context(), id, app.CreateTunnelPeerInput{
			Name: in.Name, PublicKey: in.PublicKey, ClientAddress: in.ClientIP,
		})
		if err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, p)
	})

	api.GET("/tunnels/:id/peers/:pid/config", func(c echo.Context) error {
		tid, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		pid, err := app.ParseID64(c.Param("pid"))
		if err != nil {
			return err
		}
		txt, err := tapi.PeerClientConfig(c.Request().Context(), tid, pid)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, txt)
	})

	api.GET("/tunnels/:id/peers/:pid/mikrotik", func(c echo.Context) error {
		tid, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		pid, err := app.ParseID64(c.Param("pid"))
		if err != nil {
			return err
		}
		out, err := tapi.PeerMikrotikScript(c.Request().Context(), tid, pid)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, out)
	})

	api.PUT("/tunnels/:id/peers/:pid", func(c echo.Context) error {
		tid, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		pid, err := app.ParseID64(c.Param("pid"))
		if err != nil {
			return err
		}
		var peer domain.WgPeer
		if err := c.Bind(&peer); err != nil {
			return err
		}
		peer.ID = pid
		peer.InterfaceID = tid
		return tapi.UpdateTunnelPeer(c.Request().Context(), &peer)
	})

	api.DELETE("/tunnels/:id/peers/:pid", func(c echo.Context) error {
		tid, err := app.ParseID64(c.Param("id"))
		if err != nil {
			return err
		}
		pid, err := app.ParseID64(c.Param("pid"))
		if err != nil {
			return err
		}
		return tapi.DeleteTunnelPeer(c.Request().Context(), tid, pid)
	})
}
