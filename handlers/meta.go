package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/config"
	"net/http"
)

func GetMeta(c echo.Context) error {
	name := c.Param("name")

	return c.Render(http.StatusOK, "meta.html", map[string]interface{}{
		"name":   name,
		"blocks": config.App.Meta.Blocks,
	})
}
