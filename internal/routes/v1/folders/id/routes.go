package id

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/projdocs/api/internal/middleware"
	"github.com/projdocs/api/internal/types/response"
)

func Register(r *gin.RouterGroup) {
	r.POST("/", func(c *gin.Context) {

		_, authd := c.Get(middleware.AuthenticationJWTGinContextKey)
		if !authd {
			response.Error(c, http.StatusForbidden, "authentication missing")
			return
		}

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			response.Error(c, http.StatusBadRequest, fmt.Sprintf("bad id: %v", err))
			return
		}

		response.Data(c, gin.H{"id": id})
		return
	})

}
