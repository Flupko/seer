package utils

import (
	"net/http"
	"seer/internal/config"
	"time"

	"github.com/labstack/echo/v4"
)

func GetAndClearCookie(c echo.Context, cookieName string) (string, error) {

	cookieValue, err := GetCookie(c, cookieName)

	if err != nil {
		return "", err
	}

	// Clear the state cookie
	ClearCookie(c, cookieName)

	return cookieValue, nil
}

func GetCookie(c echo.Context, cookieName string) (string, error) {

	cookie, err := c.Cookie(cookieName)

	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func ClearCookie(c echo.Context, cookieName string) {
	// Clear the state cookie
	c.SetCookie(&http.Cookie{
		Name:     cookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func SetSecureCookie(c echo.Context, name, value string, maxAge time.Duration) {
	c.SetCookie(&http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: true,
		Secure:   config.IsProduction, // Must be true in production (HTTPS)
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
	})
}
