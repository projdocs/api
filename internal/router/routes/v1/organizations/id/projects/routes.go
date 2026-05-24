package projects

import (
	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/handlers"
)

func Register(r *gin.RouterGroup) {
	r.POST("/:project-id/folders", handlers.CreateFolder)
}
