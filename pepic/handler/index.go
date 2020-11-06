package handler

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// GET /
// Index page
// Shows a form for uploading a file or entering a secret code (if configured)
func (h *PepicHandler) Index(c echo.Context) error {
	code, codeErr := h.checkSecretCode(c)
	return c.Render(http.StatusOK, "index.html", map[string]interface{}{
		"isAuthorized": codeErr == nil,
		"secretCode":   code,
	})
}
