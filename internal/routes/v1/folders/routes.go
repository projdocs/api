package folders

import (
	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/routes/v1/folders/id"
)

func Register(r *gin.RouterGroup) {
	id.Register(r.Group(":id"))
}
