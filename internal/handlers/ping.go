package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/types/response"
)

func Ping(c *gin.Context) {
	response.Data(c, gin.H{"message": "pong"})
}
