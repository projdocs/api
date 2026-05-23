package tus

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func tusBase(c *gin.Context) string {
	uploadID := extractUploadID(c)
	base := strings.TrimSuffix(c.Request.URL.Path, uploadID)
	return strings.TrimSuffix(base, "/") + "/"
}

func extractUploadID(c *gin.Context) string {
	return strings.TrimPrefix(c.Param("tuspath"), "/")
}
