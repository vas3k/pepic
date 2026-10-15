package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/jobs"
	"github.com/vas3k/pepic/pepic/processing"
	"github.com/vas3k/pepic/pepic/storage"
)

type PepicHandler struct {
	Jobs       *jobs.Queue
	Processing processing.Processing
	Storage    storage.Storage
}

const SecretCodeKey = "code"

func (h *PepicHandler) checkSecretCode(c *echo.Context) (string, error) {
	var code string

	if config.App.Global.SecretCode != "" {
		cookie, err := c.Cookie(SecretCodeKey)
		if err != nil || cookie.Value == "" {
			code = c.QueryParam(SecretCodeKey)
			if code == "" {
				if strings.HasPrefix(c.Request().Header.Get(echo.HeaderContentType), echo.MIMEMultipartForm) {
					if err := c.Request().ParseMultipartForm(config.App.Safety.MultipartMemoryBytes); err != nil {
						return "", uploadReadError(err)
					}
				}
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
		newCookie.Expires = time.Now().Add(time.Duration(config.App.Safety.CookieTTLSeconds) * time.Second)
		newCookie.HttpOnly = true
		newCookie.Path = "/"
		newCookie.SameSite = http.SameSiteLaxMode
		newCookie.Secure = c.IsTLS() || strings.HasPrefix(config.App.Global.BaseUrl, "https://")
		c.SetCookie(newCookie)
	}

	return code, nil
}
