package versions

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/projdocs/api/internal/storage"
	"github.com/projdocs/api/internal/types/response"
	"github.com/projdocs/projdocs/packages/go/database"
)

var get = func(c *gin.Context) {

	var (
		fv *database.PublicFilesVersionsSelect
		su *database.PublicStorageUploadsSelect
		sp *database.PublicStorageProvidersSelect
	)

	// load from cache
	if cacheKey := c.Request.Header.Get("Content-ID"); cacheKey == "" {
		response.Error(c, http.StatusBadRequest, `Header "Content-ID" is required`)
		return
	} else if _fv, _su, _sp, ok := cache.Get(cacheKey); !ok {
		response.Error(c, http.StatusNotFound, `Session matching Header "Content-ID" was not found`)
		return
	} else {
		fv = _fv
		su = _su
		sp = _sp
	}

	// Range: bytes=<start>-<end>
	var rangeStart, rangeEnd int64
	if rangeHeader := c.Request.Header.Get("Range"); rangeHeader == "" {
		response.Error(c, http.StatusBadRequest, `Header "Range" is required`)
		return
	} else if n, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &rangeStart, &rangeEnd); err != nil || n < 1 {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	} else if n == 1 { // client omitted end (e.g. "bytes=5242880-"), serve to EOF
		rangeEnd = fv.Size - 1
	}

	// instantiate a provider
	store, err := storage.GetProviderFrom(sp)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to construct storage provider")
		return
	}

	// get the bytes
	bytes, err := store.GetContent(c, su.ProviderId, rangeStart, rangeEnd)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to get content")
		return
	}

	// return
	chunkSize := rangeEnd - rangeStart + 1
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rangeStart, rangeEnd, fv.Size))
	c.Header("Content-Length", strconv.FormatInt(chunkSize, 10))
	c.Header("Content-Type", fv.MimeType)
	c.Header("ETag", fmt.Sprintf(`"%s"`, fv.Id))
	c.Header("Accept-Ranges", "bytes")
	c.Data(http.StatusPartialContent, fv.MimeType, bytes)
}
