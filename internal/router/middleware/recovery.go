package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/types/response"
)

// Recovery replaces gin's default recovery middleware.
// It catches panics and formats them as envelope error responses
// instead of returning a plain 500 text body.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				msg := fmt.Sprintf("internal server error: %v", rec)
				// In production you'd strip the internal detail from `msg`
				// and log it instead of sending it to the client.
				response.AbortWith(c, http.StatusInternalServerError, msg)
			}
		}()
		c.Next()
	}
}
