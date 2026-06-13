package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/projdocs/api/internal/db"
	"github.com/projdocs/api/internal/router/middleware"
	"github.com/projdocs/api/internal/storage/providers"
	"github.com/projdocs/api/internal/types/response"
)

func CreateFolder(ctx *gin.Context) {

	var parentTable string
	var parentColumnName string
	var parentId string

	parts := strings.Split(ctx.Request.URL.Path, "/")
	if len(parts) < 5 {
		response.Error(ctx, http.StatusUnprocessableEntity, "invalid path format")
		return
	}
	switch parts[4] {
	case "projects":
		parentTable = "projects"
		parentColumnName = "project_id"
		parentId = ctx.Param("project-id")
	case "organizations":
		parentTable = "organizations"
		parentColumnName = "organization_id"
		parentId = ctx.Param("organization-id")
	case "clients":
		parentTable = "clients"
		parentColumnName = "client_id"
		parentId = ctx.Param("client-id")
	case "folders":
		parentTable = "folders"
		parentColumnName = "folder_id"
		parentId = ctx.Param("folder-id")
	default:
		response.Error(ctx, http.StatusUnprocessableEntity, "unhandled for "+parts[5])
		return
	}

	// parse request
	var body struct {
		Name string `json:"name" binding:"required"`
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
	folderID := uuid.New()
	storageUploadId := uuid.New()
	var storageProviderId string

	// create the project
	// handles RLS/permissions as current user
	if _, err = txn.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO public.folders (id, name, %s, storage_upload_id) VALUES ($1, $2, $3, $4)`, parentColumnName),
		folderID.String(),
		body.Name,
		parentId,
		storageUploadId.String(),
	); err != nil {
		log.Printf("unable to create folder: %v", err)
		if strings.Index(err.Error(), "duplicate key value violates unique constraint") != -1 {
			response.Error(ctx, http.StatusConflict, fmt.Sprintf("A folder with name \"%s\" already exists", body.Name))
			return
		}
		response.Error(ctx, http.StatusInternalServerError, "failed to create folder")
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
	if err := txn.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT u.provider_id
			FROM public.storage_uploads u
			WHERE u.id = (
				SELECT t.storage_upload_id
				FROM public.%s t
				WHERE t.id = $1
			)
		`,
		parentTable,
	),
		parentId,
	).Scan(&parentFolderPath); err != nil {
		response.Error(ctx, http.StatusInternalServerError, "unable to load parent-folder id")
		return
	}

	// handle storage provider
	var store providers.Provider
	if resolved, ok := ResolveStorageProviderFromOrganization(ctx, uuid.MustParse(orgID)); !ok {
		//response is handled in the resolver
		//response.Error(ctx, http.StatusBadRequest, "storage provider not found")
		return
	} else {
		storageProviderId = resolved.Id
		storageProvider, err := providers.GetProvider(resolved)
		if err != nil {
			log.Printf("unable to get provider from storage: %v", err)
			response.Error(ctx, http.StatusInternalServerError, "unable to create storage provider")
			return
		}
		store = storageProvider
	}

	// create the folder in the storage medium
	folderPath, err := store.CreateFolder(ctx, &parentFolderPath, body.Name, map[string]string{
		"table": "folders",
		"id":    folderID.String(),
	})
	if err != nil {
		response.Error(ctx, http.StatusInternalServerError, "failed to create storage folder")
		return
	}

	// create the storage_uploads record
	if _, err = txn.ExecContext(ctx,
		`INSERT INTO public.storage_uploads (id, storage_provider_id, folder_id, provider_id) VALUES ($1, $2, $3, $4)`,
		storageUploadId.String(),
		storageProviderId,
		folderID.String(),
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

	response.Data(ctx, http.StatusCreated, gin.H{"id": folderID.String()})
}
