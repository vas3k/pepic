package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/handler"
	"github.com/vas3k/pepic/pepic/processing"
	"github.com/vas3k/pepic/pepic/storage"
	"github.com/vas3k/pepic/pepic/template"
)

func serve() error {
	if err := config.Validate(); err != nil {
		return err
	}
	limit, err := config.UploadLimit()
	if err != nil {
		return err
	}
	e := echo.New()

	h := &handler.PepicHandler{
		Processing: processing.Processing{
			Image: processing.NewImageBackend(),
			Video: processing.NewVideoBackend(),
		},
		Storage: storage.NewStorage(
			storage.NewFileSystemBackend(config.App.Storage.Dir),
		),
	}

	if err := h.OpenJobs(); err != nil {
		return err
	}
	h.Jobs.Start()

	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit(limit))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:   true,
		LogURIPath:  true,
		LogStatus:   true,
		LogLatency:  true,
		HandleError: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			c.Logger().Info("request", "method", v.Method, "path", v.URIPath, "status", v.Status, "latency", v.Latency)
			return nil
		},
	}))

	e.HTTPErrorHandler = h.ErrorHandler

	e.Renderer = template.NewTemplateRenderer("html")

	e.Static("/static", "static/")
	e.Static("/favicon", "static/favicon")
	e.File("/favicon.ico", "static/favicon/favicon.ico")

	e.GET("/", h.Index)
	e.POST("/upload/multipart/", h.UploadMultipart)
	e.POST("/upload/bytes/", h.UploadBodyBytes)
	e.POST("/upload/jobs/multipart/", h.UploadJobMultipart)
	e.POST("/upload/jobs/bytes/", h.UploadJobBytes)
	e.GET("/jobs/", h.ListJobs)
	e.GET("/jobs/:id", h.GetJob)
	e.GET("/meta/:name", h.GetMeta)
	e.GET("/:length/:name", h.GetResizedFile)
	e.GET("/:name", h.GetOriginalFile)

	address := fmt.Sprintf("%s:%d", config.App.Global.Host, config.App.Global.Port)
	server := &http.Server{
		Addr:              address,
		Handler:           e,
		ReadHeaderTimeout: time.Duration(config.App.Safety.ReadHeaderTimeoutSeconds) * time.Second,
		ReadTimeout:       time.Duration(config.App.Safety.ReadTimeoutSeconds) * time.Second,
		WriteTimeout:      time.Duration(config.App.Safety.WriteTimeoutSeconds) * time.Second,
		IdleTimeout:       time.Duration(config.App.Safety.IdleTimeoutSeconds) * time.Second,
		MaxHeaderBytes:    config.App.Safety.MaxHeaderBytes,
	}
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverError := make(chan error, 1)
	go func() { serverError <- server.ListenAndServe() }()
	select {
	case err := <-serverError:
		h.Jobs.Close()
		return err
	case <-signals.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), time.Duration(config.App.Jobs.ShutdownTimeoutSeconds)*time.Second)
	defer cancel()
	h.Jobs.Stop()
	if err := server.Shutdown(shutdown); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("HTTP shutdown: %v", err)
	}
	closed := make(chan error, 1)
	go func() { closed <- h.Jobs.Close() }()
	select {
	case err := <-closed:
		return err
	case <-shutdown.Done():
		return fmt.Errorf("shutdown deadline reached; unfinished jobs will resume on restart")
	}
}
