package mount

import (
	"net/http"
	"os"
	"strings"
	"time"

	"xray2wg/backend/internal/service"

	"github.com/labstack/echo/v4"
)

func secureCookiesFor(c echo.Context) bool {
	if c.Scheme() == "https" {
		return true
	}
	return strings.EqualFold(os.Getenv("BEHIND_HTTPS"), "true")
}

func setTokenCookies(c echo.Context, access, refresh string) {
	sec := secureCookiesFor(c)
	now := time.Now()
	aExp := int(time.Until(now.Add(service.JWTAccessTTL)).Seconds())
	if aExp < 1 {
		aExp = 1
	}
	rExp := int(time.Until(now.Add(service.JWTRefreshTTL)).Seconds())
	if rExp < 1 {
		rExp = 1
	}
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    access,
		Path:     "/api",
		MaxAge:   aExp,
		HttpOnly: true,
		Secure:   sec,
		SameSite: http.SameSiteStrictMode,
	})
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    refresh,
		Path:     "/api/v1/auth/refresh",
		MaxAge:   rExp,
		HttpOnly: true,
		Secure:   sec,
		SameSite: http.SameSiteStrictMode,
	})
}

func clearTokenCookies(c echo.Context) {
	sec := secureCookiesFor(c)
	cl := func(name, path string) {
		c.SetCookie(&http.Cookie{
			Name:     name,
			Value:    "",
			Path:     path,
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   sec,
			SameSite: http.SameSiteStrictMode,
		})
	}
	cl("access_token", "/api")
	cl("refresh_token", "/api/v1/auth/refresh")
}
