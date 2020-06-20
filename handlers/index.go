package handlers

import (
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/config"
	"net/http"
)

func Index(c echo.Context) error {
	return c.Render(http.StatusOK, "index.html", map[string]interface{}{
		"isAuthorized": CheckSecretCode(c) == nil,
	})
}

func CheckSecretCode(c echo.Context) error {
	if config.App.Global.SecretCode != "" {
		code := c.QueryParam("code")
		if code == "" {
			code = c.FormValue("code")
		}
		if code != config.App.Global.SecretCode {
			return errors.New("secret code is invalid")
		}
	}
	return nil
}
