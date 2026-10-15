package handler

import (
	"context"
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/entity"
	"github.com/vas3k/pepic/pepic/processing"
	"github.com/vas3k/pepic/pepic/utils"
)

type UploadResult struct {
	Filename string `json:"filename"`
	Url      string `json:"url"`
}

func (h *PepicHandler) UploadMultipart(c *echo.Context) error {
	release, err := limitUpload(c)
	if err != nil {
		return err
	}
	defer release()
	if _, err := h.checkSecretCode(c); err != nil {
		var httpError *echo.HTTPError
		if errors.As(err, &httpError) {
			return httpError
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "Secret code required")
	}

	err = c.Request().ParseMultipartForm(config.App.Safety.MultipartMemoryBytes)
	form := c.Request().MultipartForm
	if err != nil {
		return uploadReadError(err)
	}

	if len(form.File["media"]) > config.App.Safety.MaxUploadFiles {
		return echo.NewHTTPError(400, "Too many files")
	}
	var uploaded []UploadResult

	for _, multipartHeader := range form.File["media"] {
		log.Printf("Processing file: %s", multipartHeader.Filename)

		bytes, err := multipartToBytes(multipartHeader)
		if err != nil {
			return err
		}

		result, err := h.uploadBytes(multipartHeader.Filename, bytes)
		if err != nil {
			return err
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

func (h *PepicHandler) UploadBodyBytes(c *echo.Context) error {
	release, err := limitUpload(c)
	if err != nil {
		return err
	}
	defer release()
	if _, err := h.checkSecretCode(c); err != nil {
		var httpError *echo.HTTPError
		if errors.As(err, &httpError) {
			return httpError
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "Secret code required")
	}

	bytes, err := readUpload(c.Request().Body)
	if err != nil {
		return err
	}

	result, err := h.uploadBytes("", bytes)
	if err != nil {
		return err
	}

	return renderUploadResults([]UploadResult{
		{Filename: result.Filename, Url: result.Url()},
	}, c)
}

func (h *PepicHandler) uploadBytes(filename string, bytes []byte) (*entity.ProcessingFile, error) {
	return h.uploadBytesContext(context.Background(), filename, bytes)
}

func (h *PepicHandler) uploadBytesContext(ctx context.Context, filename string, bytes []byte) (*entity.ProcessingFile, error) {
	limit, err := config.UploadLimit()
	if err != nil {
		return nil, err
	}
	if int64(len(bytes)) > limit {
		return nil, echo.NewHTTPError(413, "Upload too large")
	}
	mime, err := processing.ValidateMediaContext(ctx, bytes)
	if err != nil {
		if errors.Is(err, processing.ErrInvalidMedia) {
			return nil, echo.NewHTTPError(415, "Unsupported, corrupt, or oversized media")
		}
		return nil, err
	}
	file := &entity.ProcessingFile{
		Filename: filename,
		Mime:     mime,
		Bytes:    bytes,
	}

	log.Printf("Processing %s file", file.Mime)

	hashedFilename, err := utils.CalculateHashName(file)
	if err != nil {
		return file, err
	}

	file.Filename = hashedFilename

	if (file.IsVideo() && !config.App.Videos.StoreOriginals) || (!file.IsVideo() && !config.App.Images.StoreOriginals) {
		if file.IsAnimated() {
			log.Printf("Converting animated image to video...")
			err = h.Processing.Video.WithContext(ctx).Convert(file, config.App.Images.GIFConvert)
			if err != nil {
				return file, err
			}
		}

		if file.Processor() == "ffmpeg" {
			log.Printf("Processing video...")

			if config.App.Videos.AutoConvert != "false" && config.App.Videos.AutoConvert != file.Mime {
				if err = h.Processing.Video.WithContext(ctx).Convert(file, config.App.Videos.AutoConvert); err != nil {
					return file, err
				}
			}
			if err = h.Processing.Video.WithContext(ctx).Transcode(file, config.App.Videos.OriginalLength); err != nil {
				return file, err
			}

		}

		if file.Processor() == "vips" {
			if err := h.Processing.Image.Process(file, config.App.Images.OriginalLength, config.App.Images.AutoConvert); err != nil {
				return file, err
			}
		}

	}

	if int64(len(file.Bytes)) > limit {
		return nil, echo.NewHTTPError(413, "Processed media too large")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
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
	return readUpload(src)
}

func renderUploadResults(results []UploadResult, c *echo.Context) error {
	accept := c.Request().Header.Get("Accept")

	if strings.HasPrefix(accept, "application/json") {
		var urls []string
		for _, result := range results {
			urls = append(urls, result.Url)
		}
		return c.JSON(http.StatusCreated, map[string]any{
			"uploaded": urls,
		})
	}

	var filenames []string
	for _, result := range results {
		filenames = append(filenames, result.Filename)
	}
	return c.Redirect(http.StatusFound, "/meta/"+strings.Join(filenames, ","))
}
