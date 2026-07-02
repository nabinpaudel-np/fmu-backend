package auth

import (
	"net/http"
	"strings"
	"time"

	"fmu-backend/internal/config"
)

const (
	AccessCookieName     = "access_token"
	RefreshCookieName    = "refresh_token"
	OAuthStateCookieName = "oauth_state"

	accessCookiePath       = "/"
	refreshCookiePath      = "/api/v1/auth"
	oauthStateCookiePath   = "/api/v1/auth/google/callback"
	oauthStateCookieMaxAge = 10 * time.Minute
)

func sameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func setCookie(w http.ResponseWriter, cfg *config.Config, name, value string, path string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   cfg.CookieDomain,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: sameSite(cfg.CookieSameSite),
	})
}

func clearCookie(w http.ResponseWriter, cfg *config.Config, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		Domain:   cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: sameSite(cfg.CookieSameSite),
	})
}

func SetAccessCookie(w http.ResponseWriter, cfg *config.Config, token string) {
	setCookie(w, cfg, AccessCookieName, token, accessCookiePath, cfg.AccessTokenExpiry)
}

func SetRefreshCookie(w http.ResponseWriter, cfg *config.Config, token string) {
	setCookie(w, cfg, RefreshCookieName, token, refreshCookiePath, cfg.RefreshTokenExpiry)
}

func SetOAuthStateCookie(w http.ResponseWriter, cfg *config.Config, state string) {
	setCookie(w, cfg, OAuthStateCookieName, state, oauthStateCookiePath, oauthStateCookieMaxAge)
}

func ClearAccessCookie(w http.ResponseWriter, cfg *config.Config) {
	clearCookie(w, cfg, AccessCookieName, accessCookiePath)
}

func ClearRefreshCookie(w http.ResponseWriter, cfg *config.Config) {
	clearCookie(w, cfg, RefreshCookieName, refreshCookiePath)
}

func ClearOAuthStateCookie(w http.ResponseWriter, cfg *config.Config) {
	clearCookie(w, cfg, OAuthStateCookieName, oauthStateCookiePath)
}

func getCookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

func GetAccessCookie(r *http.Request) string {
	return getCookie(r, AccessCookieName)
}

func GetRefreshCookie(r *http.Request) string {
	return getCookie(r, RefreshCookieName)
}

func GetOAuthStateCookie(r *http.Request) string {
	return getCookie(r, OAuthStateCookieName)
}
