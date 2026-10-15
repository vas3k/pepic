package handler

import (
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/labstack/echo/v5"
	"github.com/vas3k/pepic/pepic/config"
)

var processingBudget struct {
	sync.Mutex
	active int
}

func acquireProcessing() (func(), error) {
	processingBudget.Lock()
	defer processingBudget.Unlock()
	if processingBudget.active >= config.App.Safety.MaxConcurrentJobs {
		return nil, echo.NewHTTPError(http.StatusServiceUnavailable, "Media processing is busy; retry later")
	}
	processingBudget.active++
	return func() {
		processingBudget.Lock()
		defer processingBudget.Unlock()
		processingBudget.active--
	}, nil
}

func limitUpload(c *echo.Context) (func(), error) {
	n, err := config.UploadLimit()
	if err != nil {
		return nil, err
	}
	if c.Request().ContentLength > n {
		return nil, echo.NewHTTPError(413, "Upload too large")
	}
	release, err := acquireProcessing()
	if err != nil {
		return nil, err
	}
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, n)
	return func() {
		if c.Request().MultipartForm != nil {
			c.Request().MultipartForm.RemoveAll()
		}
		release()
	}, nil
}

func uploadReadError(err error) error {
	if echo.StatusCode(err) == http.StatusRequestEntityTooLarge {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "Upload too large")
	}
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return echo.NewHTTPError(413, "Upload too large")
	}
	return echo.NewHTTPError(400, "Could not read upload")
}

func readUpload(r io.Reader) ([]byte, error) {
	n, err := config.UploadLimit()
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(r, n+1))
	if err != nil {
		return nil, uploadReadError(err)
	}
	if int64(len(data)) > n {
		return nil, echo.NewHTTPError(413, "Upload too large")
	}
	return data, nil
}
