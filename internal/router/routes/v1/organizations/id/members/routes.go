package members

import (
	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/handlers"
)

func Register(r *gin.RouterGroup) {
	r.POST("/:member-id/folders", handlers.CreateFolder)
}
