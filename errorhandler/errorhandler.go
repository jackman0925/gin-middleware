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

// ErrorHandler returns a gin.HandlerFunc that intercepts errors attached
// to the gin context via c.Error() and formats them as JSON responses.
//
// Place it early in the chain (typically after recovery) so it can catch
// errors from all downstream handlers.
//
//	r.Use(gin.Recovery())
//	r.Use(errorhandler.ErrorHandler())
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// Only write a response if nothing has been written yet
		if c.Writer.Written() {
			return
		}

		err := c.Errors.Last().Err
		log.Errorf("request error [%s %s]: %v", c.Request.Method, c.Request.URL.Path, err)

		c.JSON(http.StatusInternalServerError, response.APIResponse{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
	}
}
