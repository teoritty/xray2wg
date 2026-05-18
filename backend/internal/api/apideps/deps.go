package apideps

import (
	"io/fs"

	"xray2wg/backend/internal/domain"
	sqldb "xray2wg/backend/internal/infrastructure/db"
	"xray2wg/backend/internal/service"
	wshub "xray2wg/backend/internal/ws"

	"github.com/labstack/echo/v4"
)

// HealthProbes is implemented by mount.Health (avoids apideps → mount import cycle).
type HealthProbes interface {
	Liveness(c echo.Context) error
	Readiness(c echo.Context) error
}

// Deps aggregates HTTP-layer dependencies for route registration.
type Deps struct {
	MasterKey   []byte
	Subs        *service.SubscriptionService
	Tunnels     *service.TunnelService
	Peers       *service.PeerService
	Auth        *service.AuthService
	Stats       domain.StatsRepository
	SubRepo     domain.SubscriptionRepository
	TunRepo     domain.TunnelRepository
	PeerRepo    domain.PeerRepository
	Set         domain.SettingRepository
	EventLog    *service.EventLog
	AuditDB     *sqldb.AuditLogRepo
	Hub         *wshub.Hub
	ManualSubID int64
	Static      fs.FS
	Health      HealthProbes
	NodeHealth  *service.NodeHealthMonitor
}
