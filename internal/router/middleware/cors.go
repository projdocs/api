package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: []string{"*"}, // tighten in production
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPatch,
			http.MethodHead,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Authorization",
			"X-Requested-With",
			// TUS protocol headers
			"Tus-Resumable",
			"Upload-Length",
			"Upload-Offset",
			"Upload-Metadata",
			"Upload-Defer-Length",
			"Upload-Concat",
			"X-HTTP-Method-Override",
		},
		ExposeHeaders: []string{
			"Location",
			"Tus-Resumable",
			"Tus-Version",
			"Tus-Max-Size",
			"Tus-Extension",
			"Upload-Offset",
			"Upload-Length",
			"Upload-Expires",
		},
		AllowCredentials: false, // must be false when AllowOrigins is "*"
		MaxAge:           12 * time.Hour,
	})
}
