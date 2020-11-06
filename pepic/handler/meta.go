package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
	"net/http"
	"strings"
)

// GET /meta/:name
// Returns HTML page with the image and its metadata
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
		"files":  files,
		"host":   c.Request().URL,
		"blocks": config.App.Meta.Blocks,
	})
}
