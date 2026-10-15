package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

type JSONError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (h *PepicHandler) ErrorHandler(c *echo.Context, err error) {
	if response, err := echo.UnwrapResponse(c.Response()); err == nil && response.Committed {
		return
	}
	code := echo.StatusCode(err)
	if code == 0 {
		code = http.StatusInternalServerError
	}
	message := http.StatusText(code)
	if code < http.StatusInternalServerError {
		var httpError *echo.HTTPError
		if errors.As(err, &httpError) {
			message = httpError.Message
		}
	}

	log.Printf("Error %d: %v", code, err)

	acceptContent := c.Request().Header.Get("Accept")
	if strings.Contains(acceptContent, "application/json") || c.Echo().Renderer == nil {
		c.JSON(code, struct {
			Error JSONError `json:"error"`
		}{
			Error: JSONError{
				Code:    code,
				Message: message,
			},
		})
	} else {
		c.Render(code, "error.html", map[string]any{
			"Code":    code,
			"Message": message,
		})
	}
}
