package id

import (
	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/uploads"
)

func Register(r *gin.RouterGroup) {
	// add tus multi-part uploads
	rg := r.Group("/upload")
	rg.Any("", uploads.TUSHandler)          // POST /v1/folders/:id/upload
	rg.Any("/*tuspath", uploads.TUSHandler) // PATCH/HEAD/DELETE /v1/folders/:id/upload/<id>
}
