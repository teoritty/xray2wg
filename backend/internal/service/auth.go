package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/service/refreshtoken"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	JWTAccessTTL  = 15 * time.Minute
	JWTRefreshTTL = 7 * 24 * time.Hour
)

// ApplianceUserID is the fixed user id for this single-admin appliance.
const ApplianceUserID int64 = 1

// AuthClaims are stable fields extracted from JWT map claims.
type AuthClaims struct {
	UserID     int64
	RefreshJTI string
}

type AuthService struct {
	settings     domain.SettingRepository
	priv         *rsa.PrivateKey
	subject      string
	refreshStore *refreshtoken.Store
}

func NewAuthService(settings domain.SettingRepository, keyPath string, _ *EventLog, parentCtx context.Context) (*AuthService, error) {
	priv, err := loadOrCreateRS256(keyPath)
	if err != nil {
		return nil, err
	}
	return &AuthService{
		settings:     settings,
		priv:         priv,
		subject:      "xray2wg",
		refreshStore: refreshtoken.NewStore(parentCtx),
	}, nil
}

func loadOrCreateRS256(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		der := x509.MarshalPKCS1PrivateKey(k)
		bl := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
		if err := os.WriteFile(path, pem.EncodeToMemory(bl), 0o600); err != nil {
			return nil, err
		}
		return k, nil
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("jwt pem decode")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func (a *AuthService) HashPassword(p string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(p), 12)
	return string(h), err
}

// IssueTokens creates a new access/refresh pair for the appliance admin user and registers refresh state.
func (a *AuthService) IssueTokens() (access, refresh string, err error) {
	return a.issuePairForUser(ApplianceUserID)
}

func (a *AuthService) issuePairForUser(userID int64) (access, refresh string, err error) {
	if a.refreshStore == nil {
		return "", "", errors.New("refresh store not configured")
	}
	jti, err := a.refreshStore.CreateSession(userID, JWTRefreshTTL)
	if err != nil {
		return "", "", err
	}
	access, refresh, err = a.signTokenPair(userID, jti)
	if err != nil {
		a.refreshStore.Revoke(jti)
		return "", "", err
	}
	return access, refresh, nil
}

// signTokenPair issues JWT access + refresh for userID embedding refreshJTI (server session must already exist).
func (a *AuthService) signTokenPair(userID int64, refreshJTI string) (access, refresh string, err error) {
	now := time.Now()
	accessClaims := jwt.MapClaims{
		"sub":     a.subject,
		"typ":     "access",
		"exp":     now.Add(JWTAccessTTL).Unix(),
		"iat":     now.Unix(),
		"user_id": float64(userID),
	}
	at := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	access, err = at.SignedString(a.priv)
	if err != nil {
		return "", "", err
	}
	refreshClaims := jwt.MapClaims{
		"sub":         a.subject,
		"typ":         "refresh",
		"exp":         now.Add(JWTRefreshTTL).Unix(),
		"iat":         now.Unix(),
		"refresh_jti": refreshJTI,
	}
	rt := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refresh, err = rt.SignedString(a.priv)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// RotateRefreshToken validates a refresh JWT and session, revokes the old session, and returns a new pair.
func (a *AuthService) RotateRefreshToken(refreshToken string) (access, refresh string, err error) {
	cl, err := a.ParseToken(refreshToken)
	if err != nil {
		return "", "", err
	}
	if typ, _ := cl["typ"].(string); typ != "refresh" {
		return "", "", domain.ErrUnauthorized
	}
	jti, ok := cl["refresh_jti"].(string)
	if !ok || jti == "" {
		return "", "", domain.ErrUnauthorized
	}
	newJTI, uid, err := a.refreshStore.Rotate(jti, JWTRefreshTTL)
	if err != nil {
		if errors.Is(err, refreshtoken.ErrSessionNotFound) || errors.Is(err, refreshtoken.ErrSessionExpired) {
			return "", "", domain.ErrUnauthorized
		}
		return "", "", err
	}
	access, refresh, err = a.signTokenPair(uid, newJTI)
	if err != nil {
		a.refreshStore.Revoke(newJTI)
		return "", "", err
	}
	return access, refresh, nil
}

// RevokeRefreshToken parses a refresh JWT and revokes its server session if present (idempotent).
func (a *AuthService) RevokeRefreshToken(refreshToken string) {
	if refreshToken == "" {
		return
	}
	cl, err := a.ParseToken(refreshToken)
	if err != nil {
		return
	}
	jti, ok := cl["refresh_jti"].(string)
	if !ok || jti == "" {
		return
	}
	a.refreshStore.Revoke(jti)
}

// ParseToken validates a JWT and returns claims.
func (a *AuthService) ParseToken(tokenStr string) (jwt.MapClaims, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected alg")
		}
		return &a.priv.PublicKey, nil
	})
	if err != nil || !tok.Valid {
		return nil, domain.ErrUnauthorized
	}
	mc, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	return mc, nil
}

// ClaimsFromAccess extracts AuthClaims from an access token's map claims.
func ClaimsFromAccess(mc jwt.MapClaims) (AuthClaims, bool) {
	if mc == nil {
		return AuthClaims{}, false
	}
	uid, ok := mc["user_id"].(float64)
	if !ok {
		return AuthClaims{}, false
	}
	return AuthClaims{UserID: int64(uid)}, true
}
