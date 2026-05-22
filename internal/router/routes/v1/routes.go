package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/router/routes/v1/folders"
	"github.com/projdocs/api/internal/types/response"
)

func Register(r *gin.RouterGroup) {
	r.GET("/ping", func(c *gin.Context) {
		response.Data(c, gin.H{"message": "pong"})
	})
	folders.Register(r.Group("/folders"))
}
