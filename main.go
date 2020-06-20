package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/vas3k/pepic/config"
	"github.com/vas3k/pepic/handlers"
	"html/template"
	"io"
)

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	if viewContext, isMap := data.(map[string]interface{}); isMap {
		viewContext["reverse"] = c.Echo().Reverse
	}
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	e := echo.New()

	// limit uploads size if needed
	if config.App.Global.MaxUploadSize != "" {
		e.Use(middleware.BodyLimit(config.App.Global.MaxUploadSize))
	}

	// json/html error handler
	e.HTTPErrorHandler = handlers.ErrorHandler

	// access logs
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${remote_ip} - [${time_rfc3339}] \"${method} ${uri}\" ${status} ${bytes_out} \"-\" \"${user_agent}\" \n",
	}))

	// templates
	e.Renderer = &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}

	// serve static files
	e.Static("/static", "static/assets")
	e.File("/favicon.ico", "static/assets/favicon/favicon.ico")

	// register routes
	e.GET("/", handlers.Index)
	e.POST("/upload/multipart/", handlers.UploadMultipart)
	e.POST("/upload/bytes/", handlers.UploadBytes)
	e.GET("/meta/:name", handlers.GetMeta)
	e.GET("/:length/:name", handlers.GetResizedFile)
	e.GET("/:name", handlers.GetOriginalFile)

	// start server
	address := fmt.Sprintf("%s:%d", config.App.Global.Host, config.App.Global.Port)
	e.Logger.Fatal(e.Start(address))
}
