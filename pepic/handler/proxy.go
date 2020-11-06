package handler

import (
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
	"net/http"
	"path"
	"strconv"
)

const MinLength = 200

// GET /:name
// Returns originally stored file
func (h *PepicHandler) GetOriginalFile(c echo.Context) error {
	file, err := h.Storage.GetFile("orig", c.Param("name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "File not found")
	}

	return h.Storage.Proxy(c, file.Path)
}

// GET /:length/:name
// Resizes the file, stores and returns it
func (h *PepicHandler) GetResizedFile(c echo.Context) error {
	lengthString := c.Param("length")
	if lengthString == "full" {
		return h.GetOriginalFile(c)
	}

	length, err := strconv.Atoi(lengthString)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Bad 'length' parameter. Need an integer!")
	}

	// return original image if requested size is bigger
	if length >= config.App.Images.OriginalLength {
		return h.GetOriginalFile(c)
	}

	// do not return too small images
	if length < MinLength {
		length = MinLength
	}

	// resize and store the resized one
	filename := c.Param("name")
	file, err := h.resizeFile(filename, length)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return h.Storage.Proxy(c, file.Path)
}

func (h *PepicHandler) resizeFile(filename string, length int) (*entity.ProcessingFile, error) {
	resizePath := path.Join("resize", strconv.Itoa(length))
	file, err := h.Storage.GetFile(resizePath, filename)
	if err == nil {
		// resized file already exists, just return it
		return file, nil
	}
	if file == nil {
		return nil, errors.New("file is empty or corrupted")
	}

	if file.IsImage() {
		if config.App.Images.LiveResize {
			err := h.Storage.ReadFileBytes(file, "orig")
			if err != nil {
				return file, err
			}

			err = h.Processing.Image.Resize(file, length)
			if err != nil {
				return file, err
			}

			err = h.Storage.StoreFile(file, resizePath)
			if err != nil {
				return file, err
			}
		}
		return file, nil
	}

	if file.IsVideo() {
		if config.App.Videos.LiveResize {
			err := h.Storage.ReadFileBytes(file, "orig")
			if err != nil {
				return file, err
			}

			err = h.Processing.Video.Transcode(file, length)
			if err != nil {
				return file, err
			}

			err = h.Storage.StoreFile(file, resizePath)
			if err != nil {
				return file, err
			}
		}
		return file, nil
	}

	return nil, errors.New("file does not exist")
}
