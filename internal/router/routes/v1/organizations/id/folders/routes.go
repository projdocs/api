package folders

import (
	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/handlers"
	"github.com/projdocs/api/internal/handlers/tus"
)

func Register(r *gin.RouterGroup) {

	// create folders in this (parent) folder
	r.POST("/:folder-id", handlers.CreateFolder)

	// add tus multi-part uploads to
	// create files in this (parent) folder
	rg := r.Group("/:folder-id/upload")
	{
		rg.Any("", tus.Handler)          // POST /v1/folders/:id/upload
		rg.Any("/*tuspath", tus.Handler) // PATCH/HEAD/DELETE /v1/folders/:id/upload/<id>
	}
}
