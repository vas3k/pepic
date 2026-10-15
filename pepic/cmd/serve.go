package cmd

import (
	"fmt"
	"net/http"
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
	return server.ListenAndServe()
}
