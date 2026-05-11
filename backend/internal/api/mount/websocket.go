package mount

import (
	"net/http"

	"xray2wg/backend/internal/api/apideps"
	"xray2wg/backend/internal/api/middleware"
	"xray2wg/backend/internal/security/origin"
	"xray2wg/backend/internal/service"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// MountWebSocket registers /api/v1/ws/stats (cookie auth, same Origin as CORS).
func MountWebSocket(auth *echo.Group, d *apideps.Deps, originCfg *origin.Config) {
	var up websocket.Upgrader
	up.CheckOrigin = func(r *http.Request) bool {
		return originCfg.AllowOrigin(r.Header.Get("Origin"))
	}

	ws := auth.Group("")
	ws.Use(middleware.WebSocketAuth(d.Auth))
	ws.GET("/ws/stats", func(c echo.Context) error {
		mc := middleware.BearerClaims(c)
		ac, ok := service.ClaimsFromAccess(mc)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		c.Set("user_id", ac.UserID)
		conn, err := up.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		d.Hub.Register(conn)
		go func() {
			defer d.Hub.Unregister(conn)
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					return
				}
			}
		}()
		return nil
	})
}
