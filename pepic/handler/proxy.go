package handler

import (
	"errors"
	"net/http"
	"path"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
	"github.com/vas3k/pepic/pepic/processing"
)

func (h *PepicHandler) GetOriginalFile(c *echo.Context) error {
	file, err := h.Storage.GetFile("orig", c.Param("name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "File not found")
	}

	return h.Storage.Proxy(c, file.Path)
}

func (h *PepicHandler) GetResizedFile(c *echo.Context) error {
	lengthString := c.Param("length")
	if lengthString == "full" {
		return h.GetOriginalFile(c)
	}

	length, err := strconv.Atoi(lengthString)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Bad 'length' parameter. Need an integer!")
	}

	if length >= config.App.Images.OriginalLength {
		return h.GetOriginalFile(c)
	}

	if length < config.App.Safety.MinResizeLength {
		length = config.App.Safety.MinResizeLength
	}

	filename := c.Param("name")
	file, err := h.resizeFile(filename, length)
	if err != nil {
		return err
	}

	return h.Storage.Proxy(c, file.Path)
}

func (h *PepicHandler) resizeFile(filename string, length int) (*entity.ProcessingFile, error) {
	resizePath := path.Join("resize", strconv.Itoa(length))
	file, err := h.Storage.GetFile(resizePath, filename)
	if err == nil {
		return file, nil
	}
	original, err := h.Storage.GetFile("orig", filename)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "File not found")
	}
	if (original.Processor() == "vips" && !config.App.Images.LiveResize) ||
		(original.Processor() == "ffmpeg" && !config.App.Videos.LiveResize) {
		return original, nil
	}
	release, err := acquireProcessing()
	if err != nil {
		return nil, err
	}
	defer release()

	if file.Processor() == "vips" {
		if config.App.Images.LiveResize {
			err := h.Storage.ReadFileBytes(file, "orig")
			if err != nil {
				return file, err
			}

			if _, err := processing.ValidateMedia(file.Bytes); err != nil {
				return nil, err
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

	if file.Processor() == "ffmpeg" {
		if config.App.Videos.LiveResize {
			err := h.Storage.ReadFileBytes(file, "orig")
			if err != nil {
				return file, err
			}

			if _, err := processing.ValidateMedia(file.Bytes); err != nil {
				return nil, err
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
