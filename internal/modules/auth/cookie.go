package auth

import (
	"errors"
	"net/http"
	"time"

	"bey/internal/config"
)

const (
	refreshTokenCookieName = "refresh_token"
	accessTokenCookieName  = "access_token"
)

func SetRefreshTokenCookie(w http.ResponseWriter, token string, expiry time.Duration, cfg *config.Config) {
	cookie := &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.App.Mode == "production",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(expiry.Seconds()),
	}
	http.SetCookie(w, cookie)
}

func GetRefreshTokenCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil {
		return "", errors.New("refresh token cookie not found")
	}
	return cookie.Value, nil
}

func DeleteRefreshTokenCookie(w http.ResponseWriter, cfg *config.Config) {
	cookie := &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.App.Mode == "production",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}

func SetAccessTokenCookie(w http.ResponseWriter, token string, expiry time.Duration, cfg *config.Config) {
	cookie := &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.App.Mode == "production",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(expiry.Seconds()),
	}
	http.SetCookie(w, cookie)
}

func GetAccessTokenCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(accessTokenCookieName)
	if err != nil {
		return "", errors.New("access token cookie not found")
	}
	return cookie.Value, nil
}

func DeleteAccessTokenCookie(w http.ResponseWriter, cfg *config.Config) {
	cookie := &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.App.Mode == "production",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}
