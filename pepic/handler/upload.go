package handler

import (
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
	"github.com/vas3k/pepic/pepic/utils"
)

type UploadResult struct {
	Filename string `json:"filename"`
	Url      string `json:"url"`
}

// POST /upload/multipart/
// Handles multipart upload
func (h *PepicHandler) UploadMultipart(c echo.Context) error {
	if _, err := h.checkSecretCode(c); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Secret code required")
	}

	form, err := c.MultipartForm()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	var uploaded []UploadResult

	for _, multipartHeader := range form.File["media"] {
		log.Printf("Processing file: %s", multipartHeader.Filename)

		bytes, err := multipartToBytes(multipartHeader)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		result, err := h.uploadBytes(multipartHeader.Filename, bytes)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}

		uploaded = append(uploaded, UploadResult{
			Filename: result.Filename,
			Url:      result.Url(),
		})
	}

	if len(uploaded) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No files to upload")
	}

	return renderUploadResults(uploaded, c)
}

// POST /upload/bytes/
// Handles raw bytes upload from body
func (h *PepicHandler) UploadBodyBytes(c echo.Context) error {
	if _, err := h.checkSecretCode(c); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Secret code required")
	}

	var bytes []byte
	if c.Request().Body != nil {
		bytes, _ = io.ReadAll(c.Request().Body)
	}

	result, err := h.uploadBytes("", bytes)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return renderUploadResults([]UploadResult{
		{Url: "/" + result.Filename},
	}, c)
}

func (h *PepicHandler) uploadBytes(filename string, bytes []byte) (*entity.ProcessingFile, error) {
	var err error

	file := &entity.ProcessingFile{
		Filename: filename,
		Mime:     utils.DetectMimeType(filename, bytes),
		Bytes:    bytes,
	}

	log.Printf("Processing %s file", file.Mime)

	hashedFilename, err := utils.CalculateHashName(file)
	if err != nil {
		return file, err
	}

	file.Filename = hashedFilename

	if !config.App.Images.StoreOriginals {
		if file.IsGIF() {
			log.Printf("Converting GIF to video...")
			err = h.Processing.Video.Convert(file, config.App.Images.GIFConvert)
			if err != nil {
				return file, err
			}
		}

		if file.IsVideo() || file.IsGIF() {
			log.Printf("Processing video...")

			err = h.Processing.Video.Transcode(file, config.App.Videos.OriginalLength)
			if err != nil {
				return file, err
			}

			if config.App.Videos.AutoConvert != "false" {
				err = h.Processing.Video.Convert(file, config.App.Videos.AutoConvert)
				if err != nil {
					return file, err
				}
			}
		}

		if file.IsImage() {
			log.Printf("Processing image...")

			err = h.Processing.Image.AutoRotate(file)
			if err != nil {
				return file, err
			}

			err = h.Processing.Image.Resize(file, config.App.Images.OriginalLength)
			if err != nil {
				return file, err
			}

			if config.App.Images.AutoConvert != "false" {
				err := h.Processing.Image.Convert(file, config.App.Images.AutoConvert)
				if err != nil {
					return file, err
				}
			}
		}
	}

	err = h.Storage.StoreFile(file, "orig")
	if err != nil {
		return file, err
	}

	return file, nil
}

func multipartToBytes(multipartFile *multipart.FileHeader) ([]byte, error) {
	src, err := multipartFile.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	return io.ReadAll(src)
}

func renderUploadResults(results []UploadResult, c echo.Context) error {
	accept := c.Request().Header.Get("Accept")

	// on json upload - return structured results
	if strings.HasPrefix(accept, "application/json") {
		var urls []string
		for _, result := range results {
			urls = append(urls, result.Url)
		}
		return c.JSON(http.StatusCreated, map[string]interface{}{
			"uploaded": urls,
		})
	}

	// on simple html upload - redirect to meta page
	var filenames []string
	for _, result := range results {
		filenames = append(filenames, result.Filename)
	}
	return c.Redirect(http.StatusFound, "/meta/"+strings.Join(filenames, ","))
}
