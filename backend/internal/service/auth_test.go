package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestIssueTokensValidPair(t *testing.T) {
	ctx := context.Background()
	a, err := testAuthService(t, ctx)
	if err != nil {
		t.Fatal(err)
	}
	access, refresh, err := a.IssueTokens()
	if err != nil {
		t.Fatal(err)
	}
	ac, err := a.ParseToken(access)
	if err != nil {
		t.Fatal(err)
	}
	if ac["typ"] != "access" {
		t.Fatalf("access typ: %v", ac["typ"])
	}
	rc, err := a.ParseToken(refresh)
	if err != nil {
		t.Fatal(err)
	}
	if rc["typ"] != "refresh" {
		t.Fatalf("refresh typ: %v", rc["typ"])
	}
	if _, ok := rc["refresh_jti"].(string); !ok {
		t.Fatalf("missing refresh_jti: %#v", rc)
	}
}

func TestRotateRefreshTokenSuccess(t *testing.T) {
	ctx := context.Background()
	a, err := testAuthService(t, ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, refresh1, err := a.IssueTokens()
	if err != nil {
		t.Fatal(err)
	}
	_, refresh2, err := a.RotateRefreshToken(refresh1)
	if err != nil {
		t.Fatal(err)
	}
	if refresh2 == refresh1 {
		t.Fatal("expected new refresh token string")
	}
	if _, _, err := a.RotateRefreshToken(refresh1); err == nil {
		t.Fatal("replay old refresh must fail")
	}
	if _, _, err := a.RotateRefreshToken(refresh2); err != nil {
		t.Fatalf("second rotation: %v", err)
	}
}

func TestDoubleRotationInvalidatesBoth(t *testing.T) {
	ctx := context.Background()
	a, err := testAuthService(t, ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, r0, err := a.IssueTokens()
	if err != nil {
		t.Fatal(err)
	}
	_, r1, err := a.RotateRefreshToken(r0)
	if err != nil {
		t.Fatal(err)
	}
	_, r2, err := a.RotateRefreshToken(r1)
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []struct{ name, tok string }{{"r0", r0}, {"r1", r1}} {
		if _, _, err := a.RotateRefreshToken(label.tok); err == nil {
			t.Fatalf("%s should be invalid", label.name)
		}
	}
	if _, _, err := a.RotateRefreshToken(r2); err != nil {
		t.Fatalf("r2 should still rotate once: %v", err)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	ctx := context.Background()
	a, err := testAuthService(t, ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, refresh, err := a.IssueTokens()
	if err != nil {
		t.Fatal(err)
	}
	a.RevokeRefreshToken(refresh)
	if _, _, err := a.RotateRefreshToken(refresh); err == nil {
		t.Fatal("expected unauthorized after revoke")
	}
}

func TestAccessTokenTTL15Minutes(t *testing.T) {
	ctx := context.Background()
	a, err := testAuthService(t, ctx)
	if err != nil {
		t.Fatal(err)
	}
	access, _, err := a.IssueTokens()
	if err != nil {
		t.Fatal(err)
	}
	mc, err := a.ParseToken(access)
	if err != nil {
		t.Fatal(err)
	}
	expF, ok := mc["exp"].(float64)
	if !ok {
		t.Fatalf("exp type %T", mc["exp"])
	}
	iatF, ok := mc["iat"].(float64)
	if !ok {
		t.Fatalf("iat type %T", mc["iat"])
	}
	got := time.Duration(int64(expF)-int64(iatF)) * time.Second
	if got != JWTAccessTTL {
		t.Fatalf("access ttl: got %v want %v", got, JWTAccessTTL)
	}
}

// memorySettings is a minimal SettingRepository for auth tests.
type memorySettings map[string]string

func (m memorySettings) Get(ctx context.Context, key string) (string, error) {
	return m[key], nil
}

func (m memorySettings) Set(ctx context.Context, key, value string) error {
	m[key] = value
	return nil
}

func testAuthService(t *testing.T, ctx context.Context) (*AuthService, error) {
	t.Helper()
	dir := t.TempDir()
	return NewAuthService(memorySettings(make(map[string]string)), filepath.Join(dir, "jwt.pem"), NewEventLog(1), ctx)
}
