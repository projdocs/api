package organizations

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/projdocs/api/internal/db"
	"github.com/projdocs/api/internal/handlers"
	"github.com/projdocs/api/internal/router/middleware"
	"github.com/projdocs/api/internal/router/routes/v1/organizations/id"
	"github.com/projdocs/api/internal/storage"
	"github.com/projdocs/api/internal/types/response"
	"github.com/projdocs/projdocs/packages/go/database"
)

func Register(r *gin.RouterGroup) {

	// admin endpoints
	admin := r.Group("")
	admin.Use(middleware.CheckRole([]string{"admin"}))
	{
		admin.POST("", createOrganization)
		admin.POST("/", createOrganization)
	}

	// individual organizations endpoints
	id.Register(r.Group("/:organization-id"))
}

func createOrganization(ctx *gin.Context) {

	// parse request
	var body struct {
		Name    string `json:"name" binding:"required"`
		Storage struct {
			Provider struct {
				Id *string `json:"id"` // optional—no binding:"required"
			} `json:"provider"`
		} `json:"storage"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		response.Error(ctx, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// get role
	role, ok := ctx.Get(middleware.AuthenticationJWTRoleGinContextKey)
	if !ok {
		response.Error(ctx, http.StatusForbidden, "invalid role")
		return
	}

	// get id
	id, ok := ctx.Get(middleware.AuthenticationJWTIDGinContextKey)
	if !ok {
		response.Error(ctx, http.StatusForbidden, "invalid id")
		return
	}

	// handle storage provider
	var s *database.PublicStorageProvidersSelect
	if body.Storage.Provider.Id == nil {
		if resolved, ok := handlers.ResolveDefaultStorageProvider(ctx); !ok {
			//response is handled in the resolver
			//response.Error(ctx, http.StatusBadRequest, "default storage provider not found")
			return
		} else {
			s = resolved
		}
	} else {
		if resolved, ok := handlers.ResolveStorageProviderFromID(ctx, uuid.MustParse(*body.Storage.Provider.Id)); !ok {
			//response is handled in the resolver
			//response.Error(ctx, http.StatusBadRequest, "storage provider not found")
			return
		} else {
			s = resolved
		}
	}

	sp, err := storage.GetProviderFrom(s)
	if err != nil {
		log.Printf("unable to get provider from storage: %v", err)
		response.Error(ctx, http.StatusInternalServerError, "unable to create storage provider")
		return
	}

	// start transaction
	txn, err := db.WithRLS(ctx, db.MustGet(), role.(string), uuid.MustParse(id.(string)))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "unable to start database transaction")
		return
	}
	defer txn.Rollback()

	// create the upload record (provider_id updated after folder is created)
	orgID := uuid.New()
	var uploadID uuid.UUID
	if err = txn.QueryRowContext(ctx,
		`INSERT INTO public.storage_uploads (provider_id, storage_provider_id, organization_id) VALUES ('', $1, $2) RETURNING id`,
		s.Id,
		orgID.String(),
	).Scan(&uploadID); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "failed to create storage upload record")
		return
	}

	// create organization
	if _, err = txn.ExecContext(ctx,
		`INSERT INTO public.organizations (id, display, storage_providers_id, storage_upload_id) VALUES ($1, $2, $3, $4)`,
		orgID.String(),
		body.Name,
		s.Id,
		uploadID,
	); err != nil {
		log.Printf("unable to create organization: %v", err)
		if strings.Index(err.Error(), "duplicate key value violates unique constraint") != -1 {
			response.Error(ctx, http.StatusConflict, fmt.Sprintf("organization with name \"%s\" already exists", body.Name))
			return
		}
		response.Error(ctx, http.StatusInternalServerError, "failed to create organization")
		return
	}

	// create the storage folder
	folderPath, err := sp.CreateFolder(ctx, nil, body.Name, map[string]string{
		"table": "organizations",
		"id":    orgID.String(),
	})

	// update the upload record with the real folder path
	if _, err = txn.ExecContext(ctx,
		`UPDATE public.storage_uploads SET provider_id = $1 WHERE id = $2`,
		folderPath,
		uploadID,
	); err != nil {
		log.Printf("unable to update storage upload record: %v", err)
		response.Error(ctx, http.StatusInternalServerError, "failed to save storage folder id")
		return
	}

	// commit
	if err = txn.Commit(); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "failed to commit changes")
		return
	}

	response.Data(ctx, http.StatusCreated, gin.H{"id": orgID.String()})
}
