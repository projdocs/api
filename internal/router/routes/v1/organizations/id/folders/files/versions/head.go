package versions

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/projdocs/api/internal/db"
	"github.com/projdocs/api/internal/router/middleware"
	"github.com/projdocs/projdocs/packages/go/database"
)

var head = func(c *gin.Context) {

	fileID, err := uuid.Parse(c.Param("file-id"))
	if err != nil {
		c.Header("X-Error", fmt.Sprintf("invalid path parameter: file-id: %s", c.Param("file-id")))
		c.Status(http.StatusBadRequest)
		return
	}

	versionID, err := uuid.Parse(c.Param("version-id"))
	if err != nil {
		c.Header("X-Error", fmt.Sprintf("invalid path parameter: version-id: %s", c.Param("version-id")))
		c.Status(http.StatusBadRequest)
		return
	}

	// get role
	role, ok := c.Get(middleware.AuthenticationJWTRoleGinContextKey)
	if !ok {
		c.Header("X-Error", "invalid role")
		c.Status(http.StatusForbidden)
		return
	}

	// get id
	id, ok := c.Get(middleware.AuthenticationJWTIDGinContextKey)
	if !ok {
		c.Header("X-Error", "invalid user id")
		c.Status(http.StatusForbidden)
		return
	}

	_db, err := db.Get()
	if err != nil {
		c.Header("X-Error", "unable to connect to database")
		c.Status(http.StatusInternalServerError)
		return
	}

	txn, err := db.WithRLS(c, _db, role.(string), uuid.MustParse(id.(string)))
	if err != nil {
		c.Header("X-Error", "unable to create a database transaction")
		c.Status(http.StatusInternalServerError)
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
		c.Header("X-Error", "unable to fetch file-version")
		c.Status(http.StatusBadRequest)
		return
	} else if fv.Id == "" || fv.Id != versionID.String() || fv.FilesId != fileID.String() {
		c.Header("X-Error", "file not found or is inaccessible")
		c.Status(http.StatusNotFound)
		return
	}

	if err := db.SetUser(txn, "admin", uuid.Nil); err != nil {
		c.Header("X-Error", "unable to set user")
		c.Status(http.StatusInternalServerError)
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
		c.Header("X-Error", "unable to fetch file-version's storage-upload record")
		c.Status(http.StatusBadRequest)
		return
	} else if su.Id == "" || su.Id != fv.StorageUploadsId {
		c.Header("X-Error", "file's upload-record not found or is inaccessible")
		c.Status(http.StatusNotFound)
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
		c.Header("X-Error", "unable to fetch file-version's storage-provider record")
		c.Status(http.StatusBadRequest)
		return
	} else if sp.Id == "" || sp.Id != su.StorageProviderId {
		c.Header("X-Error", "file's provider-record not found or is inaccessible")
		c.Status(http.StatusNotFound)
		return
	}

	cacheKey := cache.Set(&fv, &su, &sp)

	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", fv.MimeType)
	c.Header("Content-Length", strconv.FormatInt(fv.Size, 10))
	c.Header("ETag", fmt.Sprintf(`"%s"`, *su.Checksum))
	c.Header("Content-ID", cacheKey)
	c.Status(http.StatusOK)
	return
}
