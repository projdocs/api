package auth

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/db"
	"github.com/projdocs/api/internal/types/response"
)

type Provider struct {
	Id         string `json:"id"`
	Identifier string `json:"identifier"`
	Name       string `json:"display"`
}

func Register(r *gin.RouterGroup) {
	r.GET("/providers", func(c *gin.Context) {
		rows, err := db.MustGet().QueryContext(c, `SELECT p.id, p.identifier, p.name FROM auth.custom_oauth_providers p`)
		if err != nil {
			log.Printf("error querying oauth providers: %s", err.Error())
			response.Error(c, http.StatusInternalServerError, "an internal service error occurred while querying authentication providers")
			return
		}
		defer rows.Close()

		var providers []Provider
		for rows.Next() {
			var p Provider
			if err := rows.Scan(&p.Id, &p.Identifier, &p.Name); err != nil {
				log.Printf("error scanning oauth provider: %s", err.Error())
				response.Error(c, http.StatusInternalServerError, "an internal service error occurred while querying authentication providers")
				return
			}
			providers = append(providers, p)
		}
		if err := rows.Err(); err != nil {
			log.Printf("error iterating oauth providers: %s", err.Error())
			response.Error(c, http.StatusInternalServerError, "an internal service error occurred while querying authentication providers")
			return
		}

		response.Data(c, providers)
		return
	})
}
