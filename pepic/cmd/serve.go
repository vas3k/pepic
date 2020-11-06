package cmd

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/pepic/config"
	"github.com/vas3k/pepic/pepic/handler"
	"github.com/vas3k/pepic/pepic/processing"
	"github.com/vas3k/pepic/pepic/storage"
	"github.com/vas3k/pepic/pepic/template"

	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "`serve` starts server on configured port",
	RunE: func(cmd *cobra.Command, args []string) error {
		e := echo.New()

		// main app handler
		h := &handler.PepicHandler{
			Processing: processing.Processing{
				Image: processing.NewImageBackend(),
				Video: processing.NewVideoBackend(),
			},
			Storage: storage.NewStorage(
				storage.NewFileSystemBackend(config.App.Storage.Dir),
			),
		}

		// logging, limiting, panic recovery and other useful middlewares
		e.Use(middleware.Recover())
		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Format: "${remote_ip} - [${time_rfc3339}] \"${method} ${uri}\" ${status} ${bytes_out} \"-\" \"${user_agent}\" \n",
		}))

		if config.App.Global.MaxUploadSize != "" {
			// limit uploads size if needed
			e.Use(middleware.BodyLimit(config.App.Global.MaxUploadSize))
		}

		// json/html error handler
		e.HTTPErrorHandler = h.ErrorHandler

		// template
		e.Renderer = template.NewTemplateRenderer("html")

		// serve static files
		e.Static("/static", "static/")
		e.Static("/favicon", "static/favicon")
		e.File("/favicon.ico", "static/favicon/favicon.ico")

		// register routes
		e.GET("/", h.Index)
		e.POST("/upload/multipart/", h.UploadMultipart)
		e.POST("/upload/bytes/", h.UploadBodyBytes)
		e.GET("/meta/:name", h.GetMeta)
		e.GET("/:length/:name", h.GetResizedFile)
		e.GET("/:name", h.GetOriginalFile)

		// start server
		address := fmt.Sprintf("%s:%d", config.App.Global.Host, config.App.Global.Port)
		return e.Start(address)
	},
}
