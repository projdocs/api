package router

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/config"
	middleware2 "github.com/projdocs/api/internal/router/middleware"
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
	router.Use(middleware2.CORS())
	router.Use(gin.Logger())
	router.Use(middleware2.Recovery())
	router.Use(middleware2.WrapErrors())
	router.NoRoute(middleware2.NoRoute())
	router.NoMethod(middleware2.NoMethod())

	router.SetTrustedProxies(nil)

	// setup routes
	routes.Register(router.Group("/"))

	return router
}
