package tus

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/projdocs/api/internal/db"
	"github.com/projdocs/api/internal/handlers"
	"github.com/projdocs/api/internal/router/middleware"
	"github.com/projdocs/api/internal/storage"
	"github.com/projdocs/api/internal/types/response"
	"github.com/projdocs/projdocs/packages/go/database"
)

var cache = newConfigCache(1 * time.Hour)

func Handler(c *gin.Context) {

	base := tusBase(c)
	uploadID := extractUploadID(c)

	// get the folder
	folderID, err := uuid.Parse(c.Param("folder-id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("bad id: %v", err))
		return
	}

	// handle auth
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
		resolved, ok := handlers.ResolveStorageProviderFromFolder(c, folderID)
		if !ok {
			return
		}
		sp = resolved
		if uploadID != "" {
			cache.set(uploadID, sp)
		}
	}

	// instantiate StorageProvider
	s, err := storage.GetProviderFrom(sp)
	if err != nil {
		log.Printf("error creating tus handler: %v", err)
		response.Error(c, http.StatusInternalServerError, "error creating upload handler")
		return
	}

	// get parent path
	// get the storage object
	var providerID string
	if err := db.MustGet().QueryRow(
		`select u.provider_id from public.storage_uploads u where u.id = (select f.storage_upload_id from public.folders f where f.id = $1)`,
		folderID,
	).Scan(&providerID); err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to resolve parent-folder storage id")
		return
	}

	// get TUS handler
	tusHandler, err := s.ToTusHandler(uuid.MustParse(sp.Id), base, providerID)
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

	ctx := context.WithValue(c.Request.Context(), middleware.AuthenticationJWTIDGinContextKey, c.MustGet(middleware.AuthenticationJWTIDGinContextKey))
	ctx = context.WithValue(ctx, middleware.AuthenticationJWTGinContextKey, c.MustGet(middleware.AuthenticationJWTGinContextKey))
	ctx = context.WithValue(ctx, middleware.AuthenticationJWTRoleGinContextKey, c.MustGet(middleware.AuthenticationJWTRoleGinContextKey))
	stripped := http.StripPrefix(strings.TrimSuffix(base, "/"), tusHandler)
	rw := &responseInterceptor{ResponseWriter: c.Writer}
	stripped.ServeHTTP(rw, c.Request.WithContext(ctx))

	// cache on the initial create event
	if c.Request.Method == http.MethodPost && rw.uploadID != "" {
		cache.set(rw.uploadID, sp)
	}

	c.Abort()
}
