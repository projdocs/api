package versions

import (
	"github.com/gin-gonic/gin"
)

func Register(r *gin.RouterGroup) {
	vid := r.Group("/:version-id")

	//// replace an existing version
	//vid.Any("/upload", tus.MakeGinHandler(onUploadCallback))
	//vid.Any("/upload/*tuspath", tus.MakeGinHandler(onUploadCallback))

	// get an existing version
	vid.HEAD("", head)
	vid.GET("", get)
}
