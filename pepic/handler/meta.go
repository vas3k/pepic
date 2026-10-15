package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
)

func (h *PepicHandler) GetMeta(c *echo.Context) error {
	names := strings.Split(c.Param("name"), ",")
	var files []*entity.ProcessingFile

	for _, name := range names {
		file, err := h.Storage.GetFile("orig", name)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "File not found")
		}
		files = append(files, file)
	}

	return c.Render(http.StatusOK, "meta.html", map[string]any{
		"files": files,
		"host":  c.Request().URL,
		"meta":  config.App.Meta,
	})
}
