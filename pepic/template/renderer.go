package template

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/dustin/go-humanize"

	"github.com/flosch/pongo2/v6"
	"github.com/labstack/echo/v5"
	"github.com/vas3k/pepic/pepic/entity"
)

type Renderer struct {
	templates map[string]*pongo2.Template
}

func (t *Renderer) Render(c *echo.Context, w io.Writer, name string, data any) error {
	viewContext, isMap := data.(map[string]any)
	if !isMap {
		return errors.New("template context should be a map")
	}
	viewContext["reverse"] = c.Echo().Router().Routes().Reverse
	viewContext["renderFileTemplate"] = func(text string, file *entity.ProcessingFile) string {
		tpl, err := pongo2.FromString(text)
		if err != nil {
			return "ERROR"
		}
		result, _ := tpl.Execute(pongo2.Context{"file": file})
		return result
	}
	viewContext["renderMultipleFileTemplate"] = func(text string, files []*entity.ProcessingFile) string {
		tpl, err := pongo2.FromString(text)
		if err != nil {
			return "ERROR"
		}
		result, _ := tpl.Execute(pongo2.Context{"files": files})
		return result
	}
	viewContext["bytesHumanize"] = func(size int64) string { return humanize.Bytes(uint64(max(size, 0))) }
	return t.templates[name].ExecuteWriter(viewContext, w)
}

func NewTemplateRenderer(templatesPath string) *Renderer {
	templates := make(map[string]*pongo2.Template)
	filepath.Walk(templatesPath, func(file string, info os.FileInfo, err error) error {
		if path.Ext(file) == ".html" {
			templates[path.Base(file)] = pongo2.Must(pongo2.FromFile(file))
		}
		return nil
	})

	renderer := new(Renderer)
	renderer.templates = templates
	return renderer
}
