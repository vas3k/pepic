package handler

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
)

type JSONError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (h *PepicHandler) ErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	message := err.Error()
	if httpError, ok := err.(*echo.HTTPError); ok {
		code = httpError.Code
		message = fmt.Sprintf("%v", httpError.Message)
	}

	log.Printf("Error %d: %s", code, message)

	acceptContent := c.Request().Header.Get("Accept")
	if acceptContent == "application/json" {
		c.JSON(code, struct {
			error JSONError
		}{
			error: JSONError{
				Code:    code,
				Message: message,
			},
		})
	} else {
		c.Render(code, "error.html", map[string]interface{}{
			"Code":    code,
			"Message": message,
		})
	}
}
