package app

import (
	"context"
	"io"
	"strings"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/service"

	"golang.org/x/crypto/bcrypt"
)

// SettingsAPI is the use-case surface for /api/v1/settings*.
type SettingsAPI struct {
	Set  domain.SettingRepository
	Auth *service.AuthService
	Subs *service.SubscriptionService
}

func NewSettingsAPI(set domain.SettingRepository, auth *service.AuthService, subs *service.SubscriptionService) *SettingsAPI {
	return &SettingsAPI{Set: set, Auth: auth, Subs: subs}
}

func (a *SettingsAPI) GetServerHost(ctx context.Context) (string, error) {
	return a.Set.Get(ctx, "server_host")
}

func (a *SettingsAPI) SetServerHost(ctx context.Context, host string) error {
	return a.Set.Set(ctx, "server_host", strings.TrimSpace(host))
}

type PasswordChangeInput struct {
	Old string
	New string
}

func (a *SettingsAPI) ChangeAdminPassword(ctx context.Context, in PasswordChangeInput) error {
	if len(in.New) < 8 {
		return domain.ErrValidation
	}
	hash, err := a.Set.Get(ctx, "admin_password_hash")
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Old)) != nil {
		return domain.ErrInvalidPassword
	}
	nh, err := a.Auth.HashPassword(in.New)
	if err != nil {
		return err
	}
	return a.Set.Set(ctx, "admin_password_hash", nh)
}

func (a *SettingsAPI) ExportMinimal(ctx context.Context) ([]byte, error) {
	return a.Subs.ExportMinimal(ctx)
}

func (a *SettingsAPI) ImportMinimal(ctx context.Context, body []byte) error {
	return a.Subs.ImportMinimal(ctx, body)
}

// ReadLimitedBody reads at most maxBytes from r (for import uploads).
func ReadLimitedBody(r io.Reader, maxBytes int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, maxBytes))
}
