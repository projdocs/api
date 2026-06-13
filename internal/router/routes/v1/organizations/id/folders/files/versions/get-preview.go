package versions

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/projdocs/api/config"
	"github.com/projdocs/api/internal/db"
	"github.com/projdocs/api/internal/router/middleware"
	"github.com/projdocs/api/internal/storage/providers"
	"github.com/projdocs/api/internal/types/response"
	"github.com/projdocs/projdocs/packages/go/database"
)

var getPreview = func(c *gin.Context) {

	fileID, err := uuid.Parse(c.Param("file-id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("invalid path parameter: file-id: %s", c.Param("file-id")))
		return
	}

	versionID, err := uuid.Parse(c.Param("version-id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, fmt.Sprintf("invalid path parameter: version-id: %s", c.Param("version-id")))
		return
	}

	// get role
	role, ok := c.Get(middleware.AuthenticationJWTRoleGinContextKey)
	if !ok {
		response.Error(c, http.StatusForbidden, `invalid role`)
		return
	}

	// get id
	id, ok := c.Get(middleware.AuthenticationJWTIDGinContextKey)
	if !ok {
		response.Error(c, http.StatusForbidden, `invalid id`)
		return
	}

	_db, err := db.Get()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to connect to database")
		return
	}
	txn, err := db.WithRLS(c, _db, role.(string), uuid.MustParse(id.(string)))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to create a database transaction")
		return
	}
	defer txn.Rollback()

	var fv database.PublicFilesVersionsSelect
	if err := txn.QueryRowContext(c,
		`select
        created_at,
        files_id,
        id,
        last_modified_by,
        mime_type,
        number,
        size,
        storage_uploads_id
    from public.files_versions where id=$1 and files_id=$2 limit 1`,
		versionID.String(),
		fileID.String(),
	).Scan(
		&fv.CreatedAt,
		&fv.FilesId,
		&fv.Id,
		&fv.LastModifiedBy,
		&fv.MimeType,
		&fv.Number,
		&fv.Size,
		&fv.StorageUploadsId,
	); err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to fetch file-version")
		return
	} else if fv.Id == "" || fv.Id != versionID.String() || fv.FilesId != fileID.String() {
		response.Error(c, http.StatusNotFound, `file not found or is inaccessible`)
		return
	}

	// enforce max size of 5mb
	var maxSize5mb int64 = 5 * 1024 * 1024
	if fv.Size > maxSize5mb {
		response.Error(c, http.StatusBadRequest, `file size too big (max = 5mb)`)
		return
	}

	if err := db.SetUser(txn, "admin", uuid.Nil); err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to set user")
		return
	}

	var su database.PublicStorageUploadsSelect
	if err := txn.QueryRowContext(c,
		`select
        checksum,
        created_at,
        id,
        provider_id,
        storage_provider_id
    from public.storage_uploads where id=$1 limit 1`,
		fv.StorageUploadsId,
	).Scan(
		&su.Checksum,
		&su.CreatedAt,
		&su.Id,
		&su.ProviderId,
		&su.StorageProviderId,
	); err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to fetch file-version's storage-upload record")
		return
	} else if su.Id == "" || su.Id != fv.StorageUploadsId {
		response.Error(c, http.StatusNotFound, `file's upload-record not found or is inaccessible`)
		return
	}

	var sp database.PublicStorageProvidersSelect
	if err := _db.QueryRowContext(c, // query db directly for RLS purposes
		`select
        __is_migration_locked,
        created_at,
        data,
        display,
        id,
        is_valid,
        type
    from public.storage_providers
    where id=$1
    limit 1`,
		su.StorageProviderId,
	).Scan(
		&sp.IsMigrationLocked,
		&sp.CreatedAt,
		&sp.Data,
		&sp.Display,
		&sp.Id,
		&sp.IsValid,
		&sp.Type,
	); err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to fetch file-version's storage-provider record")
		return
	} else if sp.Id == "" || sp.Id != su.StorageProviderId {
		response.Error(c, http.StatusNotFound, `file's provider-record not found or is inaccessible`)
		return
	}

	storage, err := providers.GetProvider(&sp)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to instantiate storage provider")
		return
	}

	content, err := storage.GetContent(c, su.ProviderId, 0, fv.Size-1)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to fetch file-version's content")
		return
	}

	sc := config.MustGet().SafeConvert
	req, err := http.NewRequestWithContext(c, http.MethodPost, fmt.Sprintf("%s/convert", strings.TrimSuffix(sc.URL, "/")), bytes.NewReader(content))
	req.Header.Set("Authorization", "Bearer "+sc.AccessToken)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to create backend request")
		return
	}
	req.Header.Set("Content-Type", fv.MimeType)
	req.Header.Set("Content-Length", strconv.Itoa(len(content)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "generate preview failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		response.Error(c, http.StatusInternalServerError, "generate preview failed")
		return
	}

	if body, err := io.ReadAll(resp.Body); err != nil {
		response.Error(c, http.StatusInternalServerError, "generate preview failed")
		return
	} else {
		response.Data(c, body)
		return
	}
}
