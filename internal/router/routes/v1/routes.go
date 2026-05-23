package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/handlers"
	"github.com/projdocs/api/internal/router/routes/v1/clients"
	"github.com/projdocs/api/internal/router/routes/v1/folders"
	"github.com/projdocs/api/internal/router/routes/v1/members"
	"github.com/projdocs/api/internal/router/routes/v1/organizations"
	"github.com/projdocs/api/internal/router/routes/v1/projects"
)

func Register(r *gin.RouterGroup) {
	r.GET("/ping", handlers.Ping)
	organizations.Register(r.Group("/organizations"))
	clients.Register(r.Group("/clients"))
	projects.Register(r.Group("/projects"))
	members.Register(r.Group("/members"))
	folders.Register(r.Group("/folders"))
}
