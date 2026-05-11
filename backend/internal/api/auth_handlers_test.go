package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	sqldb "xray2wg/backend/internal/infrastructure/db"
	xrayinfra "xray2wg/backend/internal/infrastructure/xrayinfra"
	wginfra "xray2wg/backend/internal/infrastructure/wireguard"
	"xray2wg/backend/internal/api/mount"
	"xray2wg/backend/internal/infra"
	"xray2wg/backend/internal/service"
	wshub "xray2wg/backend/internal/ws"

	gormsqlite "github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	echo_mid "github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

func newAuthTestEcho(t *testing.T) (*echo.Echo, *Deps, context.CancelFunc) {
	t.Helper()
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://127.0.0.1")
	t.Setenv("BEHIND_HTTPS", "false")

	ctx := context.Background()
	db, err := gorm.Open(gormsqlite.Open("file:"+filepath.Join(t.TempDir(), "auth.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		x, err := db.DB()
		if err == nil {
			_ = x.Close()
		}
	})
	if err := sqldb.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	health := mount.NewHealth(sqlDB)
	health.MarkReady()

	setRepo := sqldb.NewSettingRepo(db)
	subRepo := sqldb.NewSubscriptionRepo(db)
	tunRepo := sqldb.NewTunnelRepo(db)
	peerRepo := sqldb.NewPeerRepo(db)
	statsRepo := sqldb.NewStatsRepo(db)

	el := service.NewEventLog(8)
	jwtDir := t.TempDir()
	rootCtx, cancel := context.WithCancel(context.Background())
	authSvc, err := service.NewAuthService(setRepo, filepath.Join(jwtDir, "jwt.pem"), el, rootCtx)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	subSvc := service.NewSubscriptionService(subRepo, el)
	manualID, err := subSvc.EnsureManual(ctx)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	hub := wshub.NewHub(context.Background())
	statsColl := service.NewStatsCollector(statsRepo, peerRepo, tunRepo, hub)
	xrm := xrayinfra.NewManager()
	wgm := wginfra.NewManager()
	xr := &infra.XrayAdapter{M: xrm}
	wg := &infra.WgAdapter{M: wgm}
	mk := make([]byte, 32)
	tunnelSvc := service.NewTunnelService(tunRepo, peerRepo, subRepo, xr, wg, mk, el, statsColl)
	on := func(id int64) bool { return tunnelSvc.IsRunning(id) }
	peerSvc := service.NewPeerService(peerRepo, tunRepo, mk, wg, on)

	e := echo.New()
	e.Pre(echo_mid.RemoveTrailingSlash())
	deps := &Deps{
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
		Hub:         hub,
		ManualSubID: manualID,
		Static:      nil,
		Health:      health,
	}
	if err := Register(e, deps); err != nil {
		cancel()
		t.Fatal(err)
	}
	return e, deps, cancel
}

func TestAuthCycleLoginRefreshLogout(t *testing.T) {
	e, deps, cancel := newAuthTestEcho(t)
	defer cancel()

	ctx := context.Background()
	h, err := deps.Auth.HashPassword("longpassword123")
	if err != nil {
		t.Fatal(err)
	}
	if err := deps.Set.Set(ctx, "admin_password_hash", h); err != nil {
		t.Fatal(err)
	}

	loginBody := `{"password":"longpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderOrigin, "https://127.0.0.1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("login status %d body %s", rec.Code, rec.Body.String())
	}
	var accessVal, refreshVal string
	for _, ck := range rec.Result().Cookies() {
		switch ck.Name {
		case "access_token":
			accessVal = ck.Value
		case "refresh_token":
			refreshVal = ck.Value
		}
	}
	if accessVal == "" || refreshVal == "" {
		t.Fatalf("missing cookies: %+v", rec.Result().Cookies())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	req2.Header.Set("Cookie", "access_token="+accessVal)
	req2.Header.Set(echo.HeaderOrigin, "https://127.0.0.1")
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("settings %d %s", rec2.Code, rec2.Body.String())
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req3.Header.Set("Cookie", "refresh_token="+refreshVal)
	req3.Header.Set(echo.HeaderOrigin, "https://127.0.0.1")
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusNoContent {
		t.Fatalf("refresh %d %s", rec3.Code, rec3.Body.String())
	}
	var refresh2 string
	for _, ck := range rec3.Result().Cookies() {
		if ck.Name == "refresh_token" {
			refresh2 = ck.Value
		}
	}
	if refresh2 == "" || refresh2 == refreshVal {
		t.Fatal("expected rotated refresh cookie")
	}

	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req4.Header.Set("Cookie", "refresh_token="+refreshVal)
	req4.Header.Set(echo.HeaderOrigin, "https://127.0.0.1")
	rec4 := httptest.NewRecorder()
	e.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusUnauthorized {
		t.Fatalf("replay old refresh: status %d body %s", rec4.Code, rec4.Body.String())
	}

	req5 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req5.Header.Set("Cookie", "refresh_token="+refresh2)
	req5.Header.Set(echo.HeaderOrigin, "https://127.0.0.1")
	rec5 := httptest.NewRecorder()
	e.ServeHTTP(rec5, req5)
	if rec5.Code != http.StatusNoContent {
		t.Fatalf("logout %d", rec5.Code)
	}
}

func TestAuthRefreshMissingCookieUnauthorizedJSON(t *testing.T) {
	e, _, cancel := newAuthTestEcho(t)
	defer cancel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.Header.Set(echo.HeaderOrigin, "https://127.0.0.1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("code %q", body.Error.Code)
	}
}

func seedAdminPassword(t *testing.T, deps *Deps) {
	t.Helper()
	ctx := context.Background()
	h, err := deps.Auth.HashPassword("longpassword123")
	if err != nil {
		t.Fatal(err)
	}
	if err := deps.Set.Set(ctx, "admin_password_hash", h); err != nil {
		t.Fatal(err)
	}
}

func loginAccessCookie(t *testing.T, e *echo.Echo) string {
	t.Helper()
	loginBody := `{"password":"longpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderOrigin, "https://127.0.0.1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("login status %d body %s", rec.Code, rec.Body.String())
	}
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == "access_token" {
			return ck.Value
		}
	}
	t.Fatal("missing access_token cookie")
	return ""
}

func TestAuthMeReturnsUserID(t *testing.T) {
	e, deps, cancel := newAuthTestEcho(t)
	defer cancel()
	seedAdminPassword(t, deps)
	access := loginAccessCookie(t, e)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Cookie", "access_token="+access)
	req.Header.Set(echo.HeaderOrigin, "https://127.0.0.1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.UserID != service.ApplianceUserID {
		t.Fatalf("user_id %d", body.UserID)
	}
}

func TestWebSocketUpgradeRequiresAllowedOrigin(t *testing.T) {
	e, deps, cancel := newAuthTestEcho(t)
	defer cancel()
	seedAdminPassword(t, deps)
	access := loginAccessCookie(t, e)

	srv := httptest.NewServer(e)
	defer srv.Close()
	u := strings.Replace(srv.URL, "http", "ws", 1) + "/api/v1/ws/stats"

	hdr := http.Header{}
	hdr.Set("Cookie", "access_token="+access)
	hdr.Set("Origin", "https://evil.example")
	_, _, err := websocket.DefaultDialer.Dial(u, hdr)
	if err == nil {
		t.Fatal("expected websocket dial failure for disallowed origin")
	}

	hdr2 := http.Header{}
	hdr2.Set("Cookie", "access_token="+access)
	hdr2.Set("Origin", "https://127.0.0.1")
	c, _, err := websocket.DefaultDialer.Dial(u, hdr2)
	if err != nil {
		t.Fatalf("dial with allowed origin: %v", err)
	}
	_ = c.Close()
}
