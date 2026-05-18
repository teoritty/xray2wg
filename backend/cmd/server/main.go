package main

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	apipkg "xray2wg/backend/internal/api"
	"xray2wg/backend/internal/api/mount"
	sqldb "xray2wg/backend/internal/infrastructure/db"
	"xray2wg/backend/internal/infrastructure/netconf"
	xrayinfra "xray2wg/backend/internal/infrastructure/xrayinfra"
	wginfra "xray2wg/backend/internal/infrastructure/wireguard"
	"xray2wg/backend/internal/infra"
	"xray2wg/backend/internal/service"
	tlscfg "xray2wg/backend/internal/tls"
	"xray2wg/backend/internal/telemetry"
	wshub "xray2wg/backend/internal/ws"
	"xray2wg/backend/staticfs"

	"github.com/labstack/echo/v4"
	echo_mid "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm/logger"
)

func getenvDefault(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

// tlsAutoCertEnabled reads TLS_AUTO_CERT (default true when unset).
// Set to false/0/no to require TLS_CERT_FILE+TLS_KEY_FILE when AutoCert would otherwise generate.
func tlsAutoCertEnabled() bool {
	v := strings.TrimSpace(os.Getenv("TLS_AUTO_CERT"))
	if v == "" {
		return true
	}
	return strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
}

func gormLogLevel() logger.LogLevel {
	switch getenvDefault("LOG_LEVEL", "info") {
	case "debug":
		return logger.Info
	case "warn":
		return logger.Warn
	case "error":
		return logger.Error
	default:
		return logger.Warn
	}
}

func loadMasterKey(dir string) ([]byte, error) {
	p := filepath.Join(dir, "master.key")
	b, err := os.ReadFile(p)
	if err == nil && len(b) == 32 {
		return b, nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	if err := os.WriteFile(p, k, 0o600); err != nil {
		return nil, err
	}
	return k, nil
}

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	if lvl, err := zerolog.ParseLevel(strings.TrimSpace(strings.ToLower(getenvDefault("LOG_LEVEL", "info")))); err == nil {
		zerolog.SetGlobalLevel(lvl)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	dataDir := getenvDefault("DATA_DIR", "./data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatal().Err(err).Msg("data dir")
	}

	conn, err := sqldb.InitDB(context.Background(), dataDir, gormLogLevel())
	if err != nil {
		log.Fatal().Err(err).Msg("database")
	}
	sqlDB, sqlErr := conn.DB()
	if sqlErr != nil {
		log.Fatal().Err(sqlErr).Msg("sql DB handle")
	}
	telemetry.Register(prometheus.DefaultRegisterer)
	health := mount.NewHealth(sqlDB)

	mk, err := loadMasterKey(dataDir)
	if err != nil {
		log.Fatal().Err(err).Msg("master key")
	}

	if err := netconf.EnableForwarding(); err != nil {
		log.Warn().Err(err).Msg("sysctl ip_forward (set via Docker sysctls/read-only /proc is OK on production)")
	}

	rootCtx, cancel := context.WithCancel(context.Background())

	setRepo := sqldb.NewSettingRepo(conn)
	subRepo := sqldb.NewSubscriptionRepo(conn)
	tunRepo := sqldb.NewTunnelRepo(conn)
	peerRepo := sqldb.NewPeerRepo(conn)
	statsRepo := sqldb.NewStatsRepo(conn)

	auditRepo := sqldb.NewAuditLogRepo(conn)
	el := service.NewEventLog(128)
	el.SetPersister(auditRepo)
	authSvc, err := service.NewAuthService(setRepo, filepath.Join(dataDir, "jwt_private.pem"), el, rootCtx)
	if err != nil {
		log.Fatal().Err(err).Msg("jwt")
	}

	subSvc := service.NewSubscriptionService(subRepo, el)
	manualID, err := subSvc.EnsureManual(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("manual subscription")
	}

	hub := wshub.NewHub(rootCtx)

	statsColl := service.NewStatsCollector(statsRepo, peerRepo, tunRepo, hub)
	xrm := xrayinfra.NewManager()
	wgm := wginfra.NewManager()
	xr := &infra.XrayAdapter{M: xrm}
	wg := &infra.WgAdapter{M: wgm}
	tunnelSvc := service.NewTunnelService(tunRepo, peerRepo, subRepo, xr, wg, mk, el, statsColl)

	on := func(id int64) bool { return tunnelSvc.IsRunning(id) }
	peerSvc := service.NewPeerService(peerRepo, tunRepo, mk, wg, on)

	ctx := context.Background()

	subs, err := subRepo.List(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("list subscriptions")
	}
	if err := startSubscriptionLoops(rootCtx, subs, subSvc); err != nil {
		log.Fatal().Err(err).Msg("subscription loops")
	}

	tunnelSvc.RestoreRunning(ctx, peerSvc)
	health.MarkReady()

	go statsColl.Run(rootCtx)

	nodeHealth := service.NewNodeHealthMonitor(subRepo, 60*time.Second, 3*time.Second, 10)
	go nodeHealth.Run(rootCtx)

	e := echo.New()
	e.Pre(echo_mid.RemoveTrailingSlash())
	deps := apipkg.Deps{
		MasterKey:   mk,
		Subs:        subSvc,
		Tunnels:     tunnelSvc,
		Peers:       peerSvc,
		Auth:        authSvc,
		Stats:       statsRepo,
		SubRepo:     subRepo,
		TunRepo:     tunRepo,
		PeerRepo:    peerRepo,
		Set:         setRepo,
		EventLog:    el,
		AuditDB:     auditRepo,
		Hub:         hub,
		ManualSubID: manualID,
		Static:      staticfs.Assets,
		Health:      health,
		NodeHealth:  nodeHealth,
	}
	if err := apipkg.Register(e, &deps); err != nil {
		log.Fatal().Err(err).Msg("register routes")
	}

	addr := ":" + getenvDefault("PORT", "8080")
	httpPlain := strings.EqualFold(os.Getenv("HTTP_PLAIN"), "true") || strings.EqualFold(os.Getenv("TLS_OFF"), "true")

	var httpsSrv *http.Server
	var tlsMgr *tlscfg.Manager
	if !httpPlain {
		var errTLS error
		tlsMgr, errTLS = tlscfg.NewManager(rootCtx, tlscfg.Config{
			CertFile: strings.TrimSpace(os.Getenv("TLS_CERT_FILE")),
			KeyFile:  strings.TrimSpace(os.Getenv("TLS_KEY_FILE")),
			AutoCert: tlsAutoCertEnabled(),
			DataDir:  dataDir,
		})
		if errTLS != nil {
			log.Fatal().Err(errTLS).Msg("tls manager")
		}
		httpsSrv = &http.Server{
			Addr:      addr,
			Handler:   e,
			TLSConfig: tlsMgr.TLSConfig(),
		}
		go func() {
			if err := httpsSrv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("https server")
			}
		}()
		red := strings.TrimSpace(os.Getenv("HTTP_REDIRECT_ADDR"))
		if red == "" {
			red = strings.TrimSpace(os.Getenv("TLS_REDIRECT_HTTP"))
		}
		if red != "" {
			go func() {
				h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					u := "https://" + r.Host + r.URL.RequestURI()
					http.Redirect(w, r, u, http.StatusMovedPermanently)
				})
				if err := http.ListenAndServe(red, h); err != nil {
					log.Error().Err(err).Str("addr", red).Msg("http redirect server")
				}
			}()
		}
	} else {
		go func() {
			if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("echo")
			}
		}()
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	cancel()

	shCtx, cn := context.WithTimeout(context.Background(), 15*time.Second)
	defer cn()

	log.Info().Msg("graceful shutdown: stopping tls renewal")
	if tlsMgr != nil {
		tlsMgr.Stop()
	}

	log.Info().Msg("graceful shutdown: closing websocket hub")
	if err := hub.Close(); err != nil {
		log.Error().Err(err).Msg("hub close")
	}

	log.Info().Msg("graceful shutdown: stopping tunnels")
	tunnelSvc.ShutdownAll(shCtx)

	log.Info().Msg("graceful shutdown: stopping http server")
	if httpsSrv != nil {
		if err := httpsSrv.Shutdown(shCtx); err != nil {
			log.Error().Err(err).Msg("https shutdown")
		}
	} else {
		if err := e.Shutdown(shCtx); err != nil {
			log.Error().Err(err).Msg("shutdown")
		}
	}
	log.Info().Msg("graceful shutdown: closing database")
	logTeardownError(sqlDB)

	log.Info().Msg("graceful shutdown: complete")
}
