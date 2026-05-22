package router

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/config"
	"github.com/projdocs/api/internal/middleware"
	"github.com/projdocs/api/internal/router/routes"
)

func New() *gin.Engine {

	// handle the mode
	switch config.Mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
		break
	case "release":
		gin.SetMode(gin.ReleaseMode)
		break
	default:
		log.Println("mode not recognized: " + config.Mode)
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false
	router.Use(middleware.CORS())
	router.Use(gin.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.WrapErrors())
	router.NoRoute(middleware.NoRoute())
	router.NoMethod(middleware.NoMethod())

	router.SetTrustedProxies(nil)

	// setup routes
	routes.Register(router.Group("/"))

	return router
}
