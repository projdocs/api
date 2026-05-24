package id

import (
	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/router/routes/v1/organizations/id/clients"
	"github.com/projdocs/api/internal/router/routes/v1/organizations/id/folders"
	"github.com/projdocs/api/internal/router/routes/v1/organizations/id/members"
	"github.com/projdocs/api/internal/router/routes/v1/organizations/id/projects"
)

func Register(r *gin.RouterGroup) {
	clients.Register(r.Group("/clients"))
	projects.Register(r.Group("/projects"))
	members.Register(r.Group("/members"))
	folders.Register(r.Group("/folders"))
}
