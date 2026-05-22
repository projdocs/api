package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/projdocs/api/config"
	"github.com/projdocs/api/internal/types/response"
)

const AuthenticationJWTGinContextKey = "gin:context:request:header:jwt:validated"

// Authentication returns a 401 when no user is found or when token validation fails
func Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if len(header) < 8 || header[:7] != "Bearer " {
			response.AbortWith(c, http.StatusUnauthorized, "Authorization header invalid: missing bearer token")
			return
		}

		token, err := jwt.Parse(
			[]byte(header[7:]),
			jwt.WithKeySet(config.MustGet().JWTKeys, jws.WithInferAlgorithmFromKey(true)),
			jwt.WithValidate(true),
		)
		if err != nil {
			response.AbortWith(c, http.StatusUnauthorized, fmt.Sprintf("Authorization header invalid: jwt error: %s", err.Error()))
			return
		}

		c.Set(AuthenticationJWTGinContextKey, token)
		c.Next()
	}
}
