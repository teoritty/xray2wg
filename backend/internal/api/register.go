package api

import (
	"os"
	"strings"

	"xray2wg/backend/internal/api/middleware"
	"xray2wg/backend/internal/api/mount"
	"xray2wg/backend/internal/security/origin"

	echo_mid "github.com/labstack/echo/v4/middleware"

	"github.com/labstack/echo/v4"
)

// Register wires global middleware and all HTTP/WebSocket routes.
func Register(e *echo.Echo, d *Deps) error {
	e.HTTPErrorHandler = middleware.ErrorHandler
	e.Use(echo_mid.Recover())
	e.Use(middleware.RequestID())
	e.Use(echo_mid.LoggerWithConfig(echo_mid.LoggerConfig{
		Format: `{"time":"${time_rfc3339_nano}","level":"info","request_id":"${header:X-Request-ID}","remote_ip":"${remote_ip}","method":"${method}","uri":"${uri}","status":${status},"latency":"${latency_human}","error":"${error}"}` + "\n",
	}))
	e.Use(middleware.MetricsHTTP())
	e.Use(middleware.CSP())

	originCfg, err := origin.NewConfig(strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")))
	if err != nil {
		return err
	}
	e.Use(echo_mid.CORSWithConfig(echo_mid.CORSConfig{
		AllowOriginFunc: func(origin string) (bool, error) {
			return originCfg.AllowOrigin(origin), nil
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Cookie"},
		AllowCredentials: true,
	}))

	mount.NewMetricsRegistry(os.Getenv("METRICS_API_KEY")).Mount(e)

	auth := e.Group("/api/v1")
	if d.Health != nil {
		auth.GET("/health", d.Health.Liveness)
		auth.GET("/ready", d.Health.Readiness)
	}
	mount.MountPublicAuth(auth, d)

	jwtMW := middleware.JWT(d.Auth)
	apiG := auth.Group("", jwtMW)
	mount.MountSession(apiG, d)
	mount.MountSettings(apiG, d)
	mount.MountSubscriptions(apiG, d)
	mount.MountPeers(apiG, d)
	mount.MountTunnels(apiG, d)
	mount.MountWebSocket(auth, d, originCfg)
	mount.MountAudit(apiG, d)

	if d.Static != nil {
		e.StaticFS("/", d.Static)
	}
	return nil
}
