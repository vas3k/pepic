package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/config"
	"github.com/vas3k/pepic/processing"
	"github.com/vas3k/pepic/storage"
	"net/http"
	"strconv"
)

const MinLength = 200

func GetOriginalFile(c echo.Context) error {
	file, err := processing.GetFile("orig", c.Param("name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "File not found")
	}
	return sendFile(c, file)
}

func GetResizedFile(c echo.Context) error {
	length, err := strconv.Atoi(c.Param("length"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Bad 'length' parameter. Need an integer!")
	}

	if length >= config.App.Images.OriginalLength {
		return GetOriginalFile(c)
	}

	if length < MinLength {
		length = MinLength
	}

	filename := c.Param("name")
	file, err := processing.ResizeFile(filename, length)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return sendFile(c, file)
}

func sendFile(c echo.Context, file *processing.ProcessedFile) error {
	return storage.Main.Proxy(c, file.Path)
}
