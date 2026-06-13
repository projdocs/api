package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

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
				stack := debug.Stack()
				log.Printf("panic: %v\n%s", rec, stack)
				response.AbortWith(c, http.StatusInternalServerError, "internal server error")
			}
		}()
		c.Next()
	}
}
