package projects

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
	"github.com/projdocs/api/internal/storage"
	"github.com/projdocs/api/internal/types/response"
)

func Register(r *gin.RouterGroup) {

	r.POST("", createProject)

	cid := r.Group("/:project-id")
	{
		cid.POST("/folders", handlers.CreateFolder)
	}
}

func createProject(ctx *gin.Context) {

	// parse request
	var body struct {
		Display string `json:"display" binding:"required"`
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
		response.Error(ctx, http.StatusForbidden, "invalid user id")
		return
	}

	// get org id
	orgID := ctx.Param("organization-id")
	if orgID == "" {
		response.Error(ctx, http.StatusForbidden, "invalid organization id")
		return
	}

	// start transaction
	txn, err := db.WithRLS(ctx, db.MustGet(), role.(string), uuid.MustParse(id.(string)))
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "unable to start database transaction")
		return
	}
	defer txn.Rollback()

	// vars
	projectID := uuid.New()
	storageUploadId := uuid.New()
	var storageProviderId string

	// create the project
	// handles RLS/permissions as current user
	if _, err = txn.ExecContext(ctx,
		`INSERT INTO public.projects (id, display, organization_id, storage_upload_id) VALUES ($1, $2, $3, $4)`,
		projectID.String(),
		body.Display,
		orgID,
		storageUploadId.String(),
	); err != nil {
		log.Printf("unable to create project: %v", err)
		if strings.Index(err.Error(), "duplicate key value violates unique constraint") != -1 {
			response.Error(ctx, http.StatusConflict, fmt.Sprintf("A project with name \"%s\" already exists", body.Display))
			return
		}
		response.Error(ctx, http.StatusInternalServerError, "failed to create project")
		return
	}

	// switch to admin user
	if err := db.SetUser(txn, "admin", uuid.Nil); err != nil {
		log.Printf("unable to set user: %v", err)
		response.Error(ctx, http.StatusInternalServerError, "unable to set user")
		return
	}

	// get the id for the parent folder
	var parentFolderPath string
	if err := txn.QueryRowContext(ctx, `
			SELECT u.provider_id
			FROM public.storage_uploads u
			WHERE u.id = (
				SELECT o.storage_upload_id
				FROM public.organizations o
				WHERE o.id = $1
			)
		`,
		orgID,
	).Scan(&parentFolderPath); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "unable to load parent-folder id")
		return
	}

	// handle storage provider
	var store storage.Provider
	if resolved, ok := handlers.ResolveStorageProviderFromOrganization(ctx, uuid.MustParse(orgID)); !ok {
		//response is handled in the resolver
		//response.Error(ctx, http.StatusBadRequest, "storage provider not found")
		return
	} else {
		storageProviderId = resolved.Id
		storageProvider, err := storage.GetProviderFrom(resolved)
		if err != nil {
			log.Printf("unable to get provider from storage: %v", err)
			response.Error(ctx, http.StatusInternalServerError, "unable to create storage provider")
			return
		}
		store = storageProvider
	}

	// create the folder in the storage medium
	folderPath, err := store.CreateFolder(ctx, &parentFolderPath, body.Display, map[string]string{
		"table": "projects",
		"id":    projectID.String(),
	})
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "failed to create storage folder")
		return
	}

	// create the storage_uploads record
	if _, err = txn.ExecContext(ctx,
		`INSERT INTO public.storage_uploads (id, storage_provider_id, project_id, provider_id) VALUES ($1, $2, $3, $4)`,
		storageUploadId.String(),
		storageProviderId,
		projectID.String(),
		folderPath,
	); err != nil {
		log.Printf("unable to create storage provider record: %v", err)
		response.Error(ctx, http.StatusInternalServerError, "failed to create storage upload record")
		return
	}

	// commit
	if err = txn.Commit(); err != nil {
		log.Printf("unable to commit transaction: %v", err)
		response.Error(ctx, http.StatusInternalServerError, "failed to commit changes")
		return
	}

	response.Data(ctx, http.StatusCreated, gin.H{"id": projectID.String()})

}
