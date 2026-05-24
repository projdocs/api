package tus

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
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

	folderID, err := uuid.Parse(c.Param("folder-id"))
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

	// get TUS handler
	tusHandler, err := s.ToTusHandler(base, "uploads/")
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
