package template

import (
	"errors"
	"fmt"
	"github.com/flosch/pongo2"
	"github.com/labstack/echo/v4"
	"github.com/vas3k/pepic/pepic/entity"
	"io"
	"os"
	"path"
	"path/filepath"
)

type Renderer struct {
	templates map[string]*pongo2.Template
}

func (t *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	viewContext, isMap := data.(map[string]interface{})
	if !isMap {
		return errors.New("template context should be a map")
	}
	viewContext["reverse"] = c.Echo().Reverse
	viewContext["renderFileTemplate"] = func(text string, file *entity.ProcessingFile) string {
		tpl, err := pongo2.FromString(text)
		if err != nil {
			return "ERROR"
		}
		result, _ := tpl.Execute(pongo2.Context{"file": file})
		return result
	}
	viewContext["bytesHumanize"] = func(b int64) string {
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
