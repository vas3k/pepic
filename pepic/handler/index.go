package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/vas3k/pepic/pepic/config"
)

func (h *PepicHandler) Index(c *echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	code, codeErr := h.checkSecretCode(c)
	return c.Render(http.StatusOK, "index.html", map[string]any{
		"jobPollSeconds":     config.App.Jobs.PollSeconds,
		"isAuthorized":       codeErr == nil,
		"secretCode":         code,
		"acceptedMediaTypes": strings.Join(config.App.Media.MIMETypes(), ","),
	})
}
