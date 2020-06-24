package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/config"
	"net/http"
	"time"
)

const SecretCodeKey = "code"
const SecretCodeCookieTTL = 30 * 24 * time.Hour

func Index(c echo.Context) error {
	code, codeErr := CheckSecretCode(c)
	return c.Render(http.StatusOK, "index.html", map[string]interface{}{
		"isAuthorized": codeErr == nil,
		"secretCode": code,
	})
}

func CheckSecretCode(c echo.Context) (string, error) {
	var code string
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
		c.SetCookie(newCookie)
	}
	return code, nil
}
