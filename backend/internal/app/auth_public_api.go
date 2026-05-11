package app

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/service"

	"golang.org/x/crypto/bcrypt"
)

// loginRateState is process-global (same semantics as previous mount/auth.go package vars).
var (
	loginMu       sync.Mutex
	loginAttempts = map[string][]time.Time{}
)

// PublicAuthAPI implements unauthenticated auth flows (cookies are still applied in mount).
type PublicAuthAPI struct {
	Set  domain.SettingRepository
	Auth *service.AuthService
}

func NewPublicAuthAPI(set domain.SettingRepository, auth *service.AuthService) *PublicAuthAPI {
	return &PublicAuthAPI{Set: set, Auth: auth}
}

// Login validates password and issues tokens (caller sets HttpOnly cookies).
func (a *PublicAuthAPI) Login(ctx context.Context, password, clientIP string) (access, refresh string, err error) {
	loginMu.Lock()
	now := time.Now()
	var bucket []time.Time
	for _, t := range loginAttempts[clientIP] {
		if t.After(now.Add(-time.Minute)) {
			bucket = append(bucket, t)
		}
	}
	loginAttempts[clientIP] = bucket
	blocked := len(bucket) >= 5
	loginMu.Unlock()
	if blocked {
		return "", "", domain.ErrRateLimited
	}

	hash, err := a.Set.Get(ctx, "admin_password_hash")
	if err != nil {
		return "", "", err
	}
	if hash == "" {
		return "", "", domain.ErrUnauthorized
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		loginMu.Lock()
		loginAttempts[clientIP] = append(loginAttempts[clientIP], time.Now())
		loginMu.Unlock()
		return "", "", domain.ErrInvalidPassword
	}

	loginMu.Lock()
	loginAttempts[clientIP] = nil
	loginMu.Unlock()

	return a.Auth.IssueTokens()
}

type BootstrapInput struct {
	Password string
	Confirm  string
}

// Bootstrap sets the initial admin password and issues tokens when no password exists yet.
func (a *PublicAuthAPI) Bootstrap(ctx context.Context, in BootstrapInput) (access, refresh string, err error) {
	phash, err := a.Set.Get(ctx, "admin_password_hash")
	if err != nil {
		return "", "", err
	}
	if phash != "" {
		return "", "", domain.ErrBootstrapDone
	}
	if len(in.Password) < 8 || in.Password != in.Confirm {
		return "", "", domain.ErrValidation
	}
	hash, err := a.Auth.HashPassword(in.Password)
	if err != nil {
		return "", "", err
	}
	if err := a.Set.Set(ctx, "admin_password_hash", hash); err != nil {
		return "", "", err
	}
	_ = a.Set.Set(ctx, "first_run", "false")
	return a.Auth.IssueTokens()
}

// SetupStatusJSON returns the JSON payload for GET /auth/setup-status (needs request scheme for http_warning).
func (a *PublicAuthAPI) SetupStatusJSON(ctx context.Context, requestScheme string) (map[string]any, error) {
	phash, err := a.Set.Get(ctx, "admin_password_hash")
	if err != nil {
		return nil, err
	}
	https := strings.EqualFold(os.Getenv("BEHIND_HTTPS"), "true")
	httpWarn := strings.EqualFold(os.Getenv("BEHIND_HTTPS"), "false") && requestScheme != "https"
	return map[string]any{
		"ok":           true,
		"needs_setup":  phash == "",
		"https":        https,
		"http_warning": httpWarn && phash != "",
	}, nil
}

// RotateRefresh returns new token pair or domain.ErrUnauthorized on invalid refresh.
func (a *PublicAuthAPI) RotateRefresh(refreshToken string) (access, refresh string, err error) {
	return a.Auth.RotateRefreshToken(refreshToken)
}

func (a *PublicAuthAPI) RevokeRefresh(refreshToken string) {
	a.Auth.RevokeRefreshToken(refreshToken)
}
