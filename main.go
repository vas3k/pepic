package main

import (
	"errors"
	"fmt"
	"github.com/flosch/pongo2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/vas3k/pepic/config"
	"github.com/vas3k/pepic/handlers"
	"github.com/vas3k/pepic/processing"
	"io"
	"os"
	"path"
	"path/filepath"
)

type TemplateRenderer struct {
	templates map[string]*pongo2.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	viewContext, isMap := data.(map[string]interface{})
	if !isMap {
		return errors.New("template context should be a map")
	}
	viewContext["reverse"] = c.Echo().Reverse
	viewContext["renderFileTemplate"] = func(text string, file *processing.ProcessedFile) string {
		tpl, err := pongo2.FromString(text)
		if err != nil {
			return "ERROR"
		}
		result, _ := tpl.Execute(pongo2.Context{"file": file})
		return result
	}
	viewContext["bytesHumanize"] = func (b int64) string {
		const unit = 1000
		if b < unit {
			return fmt.Sprintf("%d B", b)
		}
		div, exp := int64(unit), 0
		for n := b / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
	}
	return t.templates[name].ExecuteWriter(viewContext, w)
}

func NewTemplateRenderer(templatesPath string) *TemplateRenderer {
	templates := make(map[string]*pongo2.Template)
	filepath.Walk(templatesPath, func(file string, info os.FileInfo, err error) error {
		if path.Ext(file) == ".html" {
			templates[path.Base(file)] = pongo2.Must(pongo2.FromFile(file))
		}
		return nil
	})

	renderer := new(TemplateRenderer)
	renderer.templates = templates
	return renderer
}

func main() {
	e := echo.New()

	// json/html error handler
	e.HTTPErrorHandler = handlers.ErrorHandler

	// logging, limiting, panic recovery and other useful middlewares
	e.Use(middleware.Recover())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${remote_ip} - [${time_rfc3339}] \"${method} ${uri}\" ${status} ${bytes_out} \"-\" \"${user_agent}\" \n",
	}))
	if config.App.Global.MaxUploadSize != "" {
		// limit uploads size if needed
		e.Use(middleware.BodyLimit(config.App.Global.MaxUploadSize))
	}

	// templates
	e.Renderer = NewTemplateRenderer("templates/")

	// serve static files
	e.Static("/static", "static/")
	e.Static("/favicon", "static/favicon")
	e.File("/favicon.ico", "static/favicon/favicon.ico")

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
