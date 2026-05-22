package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body is the top-level JSON shape returned by every endpoint.
type Body struct {
	Data  any     `json:"data"`
	Error *string `json:"error"`
}

// Data writes a 200 (or any 2xx) success envelope.
//
//	response.OK(c, gin.H{"id": 42})
//	response.OK(c, http.StatusCreated, createdUser)
func Data(c *gin.Context, args ...any) {
	status := http.StatusOK
	var data any

	switch len(args) {
	case 1:
		data = args[0]
	case 2:
		if code, ok := args[0].(int); ok {
			status = code
		}
		data = args[1]
	default:
		data = nil
	}

	c.JSON(status, Body{Data: data, Error: nil})
}

// Error writes an error envelope. The data field will be null.
//
//	response.Fail(c, http.StatusNotFound, "user not found")
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, Body{Data: nil, Error: &message})
}

// AbortWith writes an error envelope and aborts the handler chain.
// Use this inside middleware or when you need to stop further processing.
//
//	response.AbortWith(c, http.StatusUnauthorized, "missing token")
func AbortWith(c *gin.Context, status int, message string) {
	c.Abort()
	c.JSON(status, Body{Data: nil, Error: &message})
}
