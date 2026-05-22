package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/middleware"
	"github.com/projdocs/api/internal/router/routes/public"
	"github.com/projdocs/api/internal/router/routes/v1"
)

func Register(r *gin.RouterGroup) {

	public.Register(r.Group("/public"))

	authed := r.Group("/")
	authed.Use(middleware.Authentication())
	{
		v1.Register(authed.Group("/v1"))
	}
}
