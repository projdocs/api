package id

import (
	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/handlers"
	"github.com/projdocs/api/internal/handlers/tus"
)

func Register(r *gin.RouterGroup) {

	// add tus multi-part uploads
	rg := r.Group("/upload")
	rg.Any("", tus.Handler)          // POST /v1/folders/:id/upload
	rg.Any("/*tuspath", tus.Handler) // PATCH/HEAD/DELETE /v1/folders/:id/upload/<id>

	// create folders in the parent folder
	r.POST("/folders", handlers.CreateFolder)
}
