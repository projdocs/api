package middleware

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/projdocs/api/config"
	"github.com/projdocs/api/internal/types/response"
)

const AuthenticationJWTGinContextKey = "gin:context:request:header:jwt"
const AuthenticationJWTRoleGinContextKey = "gin:context:request:header:jwt:role"
const AuthenticationJWTIDGinContextKey = "gin:context:request:header:jwt:id"

// Authentication returns a 401 when no user is found or when token validation fails
func Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if len(header) < 8 || header[:7] != "Bearer " {
			response.AbortWith(c, http.StatusUnauthorized, "Authorization header invalid: missing bearer token")
			return
		}

		// set token
		token, err := jwt.Parse(
			[]byte(header[7:]),
			jwt.WithKeySet(config.MustGet().JWTKeys, jws.WithInferAlgorithmFromKey(true)),
			jwt.WithValidate(true),
		)
		if err != nil {
			response.AbortWith(c, http.StatusUnauthorized, fmt.Sprintf("Authorization header invalid: jwt error: %s", err.Error()))
			return
		} else {
			c.Set(AuthenticationJWTGinContextKey, token)
		}

		// set role
		if role, ok := token.Get("role"); !ok {
			response.AbortWith(c, http.StatusUnauthorized, "Authorization header invalid: missing role")
			return
		} else {
			c.Set(AuthenticationJWTRoleGinContextKey, role)
		}

		// set id
		if id, err := uuid.Parse(token.Subject()); err != nil {
			response.AbortWith(c, http.StatusBadRequest, "Authorization header invalid: invalid subject")
			return
		} else {
			c.Set(AuthenticationJWTIDGinContextKey, id.String())
		}

		// done
		c.Next()
	}
}

var validPostgresRole = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z_]{0,62}$`)

func CheckRole(allowed []string) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		// retrieve role from up-stream processing
		role, ok := ctx.Get(AuthenticationJWTRoleGinContextKey)
		if !ok {
			response.AbortWith(ctx, http.StatusForbidden, "invalid role")
			return
		}

		// sanitize
		if !validPostgresRole.MatchString(role.(string)) {
			response.AbortWith(ctx, http.StatusForbidden, "invalid role")
			return
		}

		// check the input role
		found := false
		for _, allowable := range allowed {
			if role == allowable {
				found = true
				break
			}
		}
		if !found {
			response.AbortWith(ctx, http.StatusForbidden, "forbidden")
			return
		}

		// ok
		ctx.Next()
		return
	}
}
