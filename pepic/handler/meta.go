package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
)

// GET /meta/:name
// Render HTML page with uploaded images/videos and pre-defined templates for them
func (h *PepicHandler) GetMeta(c echo.Context) error {
	names := strings.Split(c.Param("name"), ",")
	var files []*entity.ProcessingFile

	for _, name := range names {
		file, err := h.Storage.GetFile("orig", name)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "File not found")
		}
		files = append(files, file)
	}

	return c.Render(http.StatusOK, "meta.html", map[string]interface{}{
		"files": files,
		"host":  c.Request().URL,
		"meta":  config.App.Meta,
	})
}
