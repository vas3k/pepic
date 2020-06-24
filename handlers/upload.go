package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/processing"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
)

type UploadResult struct {
	Filename string `json:"filename"`
	Url string `json:"url"`
}

func UploadMultipart(c echo.Context) error {
	if _, err := CheckSecretCode(c); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Secret code required")
	}

	form, err := c.MultipartForm()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	var uploaded []UploadResult

	for _, multipartHeader := range form.File["media"] {
		log.Printf("Processing file: %s", multipartHeader.Filename)

		data, err := multipartToBytes(multipartHeader)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		result, err := processing.UploadFile(multipartHeader.Filename, data)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}

		uploaded = append(uploaded, UploadResult{
			Filename: result.Filename,
			Url: result.Url(),
		})
	}

	if len(uploaded) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No files to upload")
	}

	return renderResults(uploaded, c)
}

func multipartToBytes(multipartFile *multipart.FileHeader) ([]byte, error) {
	src, err := multipartFile.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	return ioutil.ReadAll(src)
}

func UploadBytes(c echo.Context) error {
	if _, err := CheckSecretCode(c); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Secret code required")
	}

	var body []byte
	if c.Request().Body != nil {
		body, _ = ioutil.ReadAll(c.Request().Body)
	}

	result, err := processing.UploadFile("", body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return renderResults([]UploadResult{
		{Url: "/" + result.Filename},
	}, c)
}

func renderResults(results []UploadResult, c echo.Context) error {
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
	return c.Redirect(http.StatusFound, "/meta/" + strings.Join(filenames, ","))
}
