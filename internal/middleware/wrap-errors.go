package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/types/response"
)

// WrapErrors re-wraps any response whose body was written with a
// raw status code (e.g. c.AbortWithStatus(401)) so it still conforms
// to the envelope shape. It runs *after* handlers via c.Next().
//
// Note: if a handler already wrote JSON via response.Data / response.Error
// this middleware is a no-op—it only acts when the response body is empty.
func WrapErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only intervene if the handler wrote no body at all.
		if c.Writer.Written() && c.Writer.Size() == 0 {
			status := c.Writer.Status()
			if status >= 400 {
				msg := http.StatusText(status)
				if msg == "" {
					msg = "unknown error"
				}
				// Re-write the status + body. Because headers are already
				// sent at this point only when using streaming; for standard
				// JSON responses this is safe.
				c.JSON(status, response.Body{Data: nil, Error: &msg})
			}
		}
	}
}
