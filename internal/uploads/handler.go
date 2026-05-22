package uploads

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/projdocs/api/config"
	"github.com/projdocs/api/internal/db"
	"github.com/projdocs/api/internal/middleware"
	"github.com/projdocs/api/internal/types/response"
	"github.com/projdocs/api/internal/uploads/handlers"
	"github.com/projdocs/projdocs/packages/go/database"
	"github.com/tus/tusd/v2/pkg/handler"
)

var cache = newConfigCache(1 * time.Hour)

func TUSHandler(c *gin.Context) {

	base := tusBase(c)
	uploadID := extractUploadID(c)

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("bad id: %v", err))
		return
	}

	auth, authd := c.Get(middleware.AuthenticationJWTGinContextKey)
	if !authd {
		response.Error(c, http.StatusForbidden, "authentication missing")
		return
	}
	token, ok := auth.(jwt.Token)
	if !ok {
		response.Error(c, http.StatusForbidden, "invalid authentication token")
		return
	}

	if _, err := uuid.Parse(token.Subject()); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid subject")
		return
	}

	var sp *database.PublicStorageProvidersSelect
	if uploadID != "" {
		sp = cache.get(uploadID)
	}
	if sp == nil {
		resolved, ok := resolveStorageProvider(c, folderID)
		if !ok {
			return
		}
		sp = resolved
		if uploadID != "" {
			cache.set(uploadID, sp)
		}
	}

	tusHandler, err := buildTUSHandler(base, sp)
	if err != nil {
		log.Printf("error creating tus handler: %v", err)
		response.Error(c, http.StatusInternalServerError, "error creating upload handler")
		return
	}

	// drain completion events for all request types
	go func() {
		for event := range tusHandler.CompleteUploads {
			log.Printf("upload complete: id=%s size=%d", event.Upload.ID, event.Upload.Size)
			cache.delete(event.Upload.ID)
		}
	}()

	stripped := http.StripPrefix(strings.TrimSuffix(base, "/"), tusHandler)

	if c.Request.Method == http.MethodPost {
		rw := &responseInterceptor{ResponseWriter: c.Writer}
		stripped.ServeHTTP(rw, c.Request)
		if rw.uploadID != "" {
			cache.set(rw.uploadID, sp)
		}
		c.Abort()
		return
	}

	stripped.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}

func tusBase(c *gin.Context) string {
	uploadID := extractUploadID(c)
	base := strings.TrimSuffix(c.Request.URL.Path, uploadID)
	return strings.TrimSuffix(base, "/") + "/"
}

func buildTUSHandler(base string, sp *database.PublicStorageProvidersSelect) (*handler.Handler, error) {
	cfg := config.MustGet()

	switch sp.Type {
	case "BUILT_IN":
		return handlers.NewS3TUSHandler(base, handlers.S3Config{
			Region:          "local",
			Bucket:          "projdocs",
			AccessKeyID:     cfg.S3.AccessKey,
			SecretAccessKey: cfg.S3.SecretKey,
			Endpoint:        fmt.Sprintf("%s/storage/v1/s3", cfg.KongURL),
		})
	default:
		return nil, fmt.Errorf("api not configured for '%s' storage provider", sp.Type)
	}
}

func extractUploadID(c *gin.Context) string {
	return strings.TrimPrefix(c.Param("tuspath"), "/")
}

func resolveStorageProvider(c *gin.Context, folderID uuid.UUID) (*database.PublicStorageProvidersSelect, bool) {
	var orgID uuid.UUID
	err := db.MustGet().QueryRowContext(c, `SELECT private.get_folder_organization_id($1)`, folderID.String()).Scan(&orgID)
	if err != nil {
		log.Printf("error getting folder's organization ID: %v", err)
		response.Error(c, http.StatusInternalServerError, "error getting folder's organization ID")
		return nil, false
	}

	var sp database.PublicStorageProvidersSelect
	err = db.MustGet().QueryRowContext(c, `
		SELECT sp.__is_migration_locked, sp.created_at, sp.data, sp.id, sp.is_valid, sp.type
		FROM public.storage_providers sp
		WHERE sp.id = (
			SELECT o.storage_providers_id
			FROM public.organizations o
			WHERE o.id = $1
			LIMIT 1
		)
		LIMIT 1
	`, orgID).Scan(
		&sp.IsMigrationLocked,
		&sp.CreatedAt,
		&sp.Data,
		&sp.Id,
		&sp.IsValid,
		&sp.Type,
	)
	if errors.Is(err, sql.ErrNoRows) {
		response.Error(c, http.StatusNotFound, "storage provider not found")
		return nil, false
	} else if err != nil {
		log.Printf("error getting storage provider: %v", err)
		response.Error(c, http.StatusInternalServerError, "error getting storage provider")
		return nil, false
	}

	if !sp.IsValid {
		response.Error(c, http.StatusBadRequest, "storage provider not properly configured for this organization")
		return nil, false
	}

	return &sp, true
}
