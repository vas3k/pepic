package handler

import (
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/processing"
	"github.com/vas3k/pepic/pepic/storage"
	"net/http"
	"time"
)

type PepicHandler struct {
	Processing processing.Processing
	Storage    storage.Storage
}

const SecretCodeKey = "code"
const SecretCodeCookieTTL = 30 * 24 * time.Hour

// Auth for poor people
// We simply store secret code in http-only cookies. Works for us.
// You'd better think about extra protection layer on top.
// API gateway or nginx + basic auth will suffice.
func (h *PepicHandler) checkSecretCode(c echo.Context) (string, error) {
	var code string

	// ignore code check if it's not configured
	if config.App.Global.SecretCode != "" {
		cookie, err := c.Cookie(SecretCodeKey)
		if err != nil || cookie.Value == "" {
			code = c.QueryParam(SecretCodeKey)
			if code == "" {
				code = c.FormValue(SecretCodeKey)
			}
		} else {
			code = cookie.Value
		}

		if code != config.App.Global.SecretCode {
			return code, errors.New("secret code is invalid")
		}

		newCookie := new(http.Cookie)
		newCookie.Name = SecretCodeKey
		newCookie.Value = code
		newCookie.Expires = time.Now().Add(SecretCodeCookieTTL)
		newCookie.HttpOnly = true
		c.SetCookie(newCookie)
	}

	return code, nil
}
