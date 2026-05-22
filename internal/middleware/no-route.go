package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/types/response"
)

// NoRoute returns a 404 envelope (register via engine.NoRoute).
func NoRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, "route not found")
	}
}
