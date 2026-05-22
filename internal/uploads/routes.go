package uploads

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/types/response"
)

// Package-level cache; lives for the lifetime of the process.
var cache = newConfigCache(24 * time.Hour)

func Register(rg *gin.RouterGroup) {
	rg.Any("/*tuspath", func(c *gin.Context) {

		uploadID := extractUploadID(c)

		var cfg config
		if uploadID != "" {
			if cached, ok := cache.get(uploadID); ok {
				cfg = cached
			}
		}

		if (cfg == S3Config{}) {
			_cfg, err := resolveS3Config(c)
			if err != nil {
				response.AbortWith(c, http.StatusBadRequest, "could not resolve S3 config: "+err.Error())
				return
			}
			cfg = _cfg
		}

		// ── 3. Build tusd handler ─────────────────────────────────────────────
		tusHandler, err := newTUSHandler(rg.BasePath(), cfg)
		if err != nil {
			response.AbortWith(c, http.StatusInternalServerError, "failed to initialise upload handler")
			return
		}

		// ── 4. On completion: cache the config under the new ID, then clean up ─
		go func() {
			for event := range tusHandler.CompleteUploads {
				log.Printf("upload complete: id=%s size=%d bucket=%s",
					event.Upload.ID, event.Upload.Size, cfg.Bucket)

				// Cache under the upload ID so subsequent PATCH/HEAD hits are free.
				cache.set(event.Upload.ID, cfg)

				// Remove on completion—upload is done, no more requests expected.
				cache.delete(event.Upload.ID)
			}
		}()

		// ── 5. Also cache after POST so the very next PATCH is a cache hit ────
		// tusd sets Upload-Location: <basePath><uploadID> on the response.
		// We intercept the response writer to capture it.
		rw := &responseInterceptor{ResponseWriter: c.Writer}
		http.StripPrefix("/v1/upload", tusHandler).ServeHTTP(rw, c.Request)

		if c.Request.Method == http.MethodPost && rw.uploadID != "" {
			cache.set(rw.uploadID, cfg)
		}

		c.Abort()
	})
}

// extractUploadID pulls the upload ID from the URL path.
// TUS URLs are: /v1/upload/<uploadID>
func extractUploadID(c *gin.Context) string {
	path := strings.TrimPrefix(c.Param("tuspath"), "/")
	return path
}
