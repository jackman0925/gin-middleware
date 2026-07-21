// Package errorhandler provides a gin middleware that intercepts errors
// pushed to gin context via c.Error() and formats them as JSON responses.
//
// Use this as a catch-all: downstream handlers can attach errors to the
// context without writing a response, and this middleware will format
// and return them at the end of the chain.
package errorhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackman0925/gin-middleware/log"
	"github.com/jackman0925/gin-middleware/response"
)

const defaultErrorMessage = "internal server error"

// ErrorMapper maps a Gin error to an HTTP status and a client-safe message.
// Returning an invalid HTTP error status falls back to status 500.
type ErrorMapper func(*gin.Error) (status int, message string)

// ErrorHandler returns a gin.HandlerFunc that intercepts errors attached
// to the gin context via c.Error() and formats them as JSON responses.
//
// Place it early in the chain (typically after recovery) so it can catch
// errors from all downstream handlers.
//
//	r.Use(gin.Recovery())
//	r.Use(errorhandler.ErrorHandler())
func ErrorHandler() gin.HandlerFunc {
	return ErrorHandlerWithMapper(nil)
}

// ErrorHandlerWithMapper returns an error handler with application-specific
// status and public-message mapping. Without a mapper, private errors receive a
// generic message and errors marked gin.ErrorTypePublic expose their message.
func ErrorHandlerWithMapper(mapper ErrorMapper) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		ginErr := c.Errors.Last()
		log.Errorf("request error [%s %s]: %v", c.Request.Method, c.Request.URL.Path, ginErr.Err)

		// Only write a response if nothing has been written yet. Errors are still
		// logged above so a partially written response does not hide failures.
		if c.Writer.Written() {
			return
		}

		status := http.StatusInternalServerError
		message := defaultErrorMessage
		if mapper != nil {
			status, message = mapper(ginErr)
			if status < 400 || status > 599 {
				status = http.StatusInternalServerError
			}
			if message == "" {
				message = defaultErrorMessage
			}
		} else if ginErr.IsType(gin.ErrorTypePublic) {
			message = ginErr.Error()
		}

		c.JSON(status, response.APIResponse{
			Code:    status,
			Message: message,
		})
	}
}
