package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/config"
	"github.com/vas3k/pepic/processing"
	"net/http"
	"strings"
)

func GetMeta(c echo.Context) error {
	names := strings.Split(c.Param("name"), ",")
	var files []*processing.ProcessedFile

	for _, name := range names {
		file, err := processing.GetFile("orig", name)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "File not found")
		}
		files = append(files, file)
	}

	return c.Render(http.StatusOK, "meta.html", map[string]interface{}{
		"files": files,
		"host": c.Request().URL,
		"blocks": config.App.Meta.Blocks,
	})
}
