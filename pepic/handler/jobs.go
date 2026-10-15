package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"

	"github.com/labstack/echo/v5"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/jobs"
	"github.com/vas3k/pepic/pepic/processing"
)

type jobResponse struct {
	jobs.Job
	StatusURL   string `json:"status_url"`
	PollSeconds int    `json:"poll_after_seconds"`
}

func (h *PepicHandler) OpenJobs() error {
	limit, err := config.UploadLimit()
	if err != nil {
		return err
	}
	dir := config.App.Jobs.Dir
	if dir == "" {
		dir = filepath.Join(config.App.Storage.Dir, ".jobs")
	}
	h.Jobs, err = jobs.Open(dir, config.App.Jobs, limit, config.App.Safety.MaxUploadFiles, func(ctx context.Context, data []byte) (jobs.Result, error) {
		file, err := h.uploadBytesContext(processing.WithJobContext(ctx), "", data)
		if err != nil {
			return jobs.Result{}, err
		}
		return jobs.Result{Filename: file.Filename, URL: file.Url(), MIME: file.Mime}, nil
	}, acquireProcessing)
	return err
}

func (h *PepicHandler) UploadJobBytes(c *echo.Context) error     { return h.uploadJob(c, false) }
func (h *PepicHandler) UploadJobMultipart(c *echo.Context) error { return h.uploadJob(c, true) }

func (h *PepicHandler) uploadJob(c *echo.Context, multipart bool) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	release, err := limitUpload(c)
	if err != nil {
		return err
	}
	defer release()
	if _, err = h.checkSecretCode(c); err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return err
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "Secret code required")
	}
	if h.Jobs == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Background uploads unavailable")
	}
	inputs := []io.Reader{c.Request().Body}
	var names []string
	if multipart {
		if err = c.Request().ParseMultipartForm(config.App.Safety.MultipartMemoryBytes); err != nil {
			return uploadReadError(err)
		}
		headers := c.Request().MultipartForm.File["media"]
		if len(headers) == 0 || len(headers) > config.App.Safety.MaxUploadFiles {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid number of files")
		}
		inputs = nil
		for _, header := range headers {
			file, err := header.Open()
			if err != nil {
				return err
			}
			defer file.Close()
			inputs = append(inputs, file)
			names = append(names, header.Filename)
		}
	}
	job, err := h.Jobs.Submit(inputs, names...)
	if errors.Is(err, jobs.ErrFull) {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Background upload queue is full; retry later")
	}
	if errors.Is(err, jobs.ErrTooLarge) {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "Upload too large")
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) || echo.StatusCode(err) == http.StatusRequestEntityTooLarge {
			return uploadReadError(err)
		}
		return err
	}
	response := jobResponse{Job: job, StatusURL: "/jobs/" + job.ID, PollSeconds: config.App.Jobs.PollSeconds}
	c.Response().Header().Set("Location", response.StatusURL)
	return c.JSON(http.StatusAccepted, response)
}

func (h *PepicHandler) GetJob(c *echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	if _, err := h.checkSecretCode(c); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Secret code required")
	}
	if h.Jobs == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Background uploads unavailable")
	}
	job, err := h.Jobs.Get(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Job not found or expired")
	}
	return c.JSON(http.StatusOK, jobResponse{Job: job, StatusURL: "/jobs/" + job.ID, PollSeconds: config.App.Jobs.PollSeconds})
}

func (h *PepicHandler) ListJobs(c *echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	if _, err := h.checkSecretCode(c); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Secret code required")
	}
	if h.Jobs == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Background uploads unavailable")
	}
	return c.JSON(http.StatusOK, struct {
		Jobs        []jobs.Job `json:"jobs"`
		PollSeconds int        `json:"poll_after_seconds"`
	}{Jobs: h.Jobs.List(), PollSeconds: config.App.Jobs.PollSeconds})
}
