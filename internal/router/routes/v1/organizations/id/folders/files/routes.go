package files

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/projdocs/api/internal/db"
	"github.com/projdocs/api/internal/handlers/tus"
	"github.com/projdocs/api/internal/router/middleware"
	"github.com/projdocs/api/internal/storage"
	"github.com/tus/tusd/v2/pkg/handler"
)

func Register(r *gin.RouterGroup) {
	fid := r.Group("/:file-id")

	// create a new version
	fid.Any("/upload", tus.MakeGinHandler(onUploadCallback))
	fid.Any("/upload/*tuspath", tus.MakeGinHandler(onUploadCallback))
}

var onUploadCallback storage.Callback = func(
	storageProviderID uuid.UUID,
	basePath string,
	parent string,
	checksum string,
	hook handler.HookEvent,
) handler.HTTPResponse {

	folderID := strings.Split(hook.HTTPRequest.URI, "/")[5]
	fileID := strings.Split(hook.HTTPRequest.URI, "/")[7]

	// get db connection
	var pg *sql.DB
	if _pg, err := db.Get(); err != nil {
		return handler.HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Body:       `{"error":"unable to connect to database","data":null}`,
			Header: handler.HTTPHeader{
				"Content-Type": "application/json",
			},
		}
	} else {
		pg = _pg
	}

	// create transaction
	var txn *sql.Tx
	if _txn, err := pg.BeginTx(context.Background(), nil); err != nil {
		return handler.HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Body:       `{"error":"unable to begin database transaction","data":null}`,
			Header: handler.HTTPHeader{
				"Content-Type": "application/json",
			},
		}
	} else {
		txn = _txn
	}
	defer txn.Rollback()

	if err := db.SetUser(txn, hook.Context.Value(middleware.AuthenticationJWTRoleGinContextKey).(string), uuid.MustParse(hook.Context.Value(middleware.AuthenticationJWTIDGinContextKey).(string))); err != nil {
		return handler.HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Body:       `{"error":"failed to handle authentication","data":null}`,
			Header: handler.HTTPHeader{
				"Content-Type": "application/json",
			},
		}
	}

	// get the provider ID
	var providerID string
	if err := db.MustGet().QueryRow(
		`select u.provider_id from public.storage_uploads u where u.id = (select f.storage_upload_id from public.folders f where f.id = $1)`,
		folderID,
	).Scan(&providerID); err != nil {
		return handler.HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Body:       `{"error":"unable to resolve parent-folder storage ID","data":null}`,
			Header: handler.HTTPHeader{
				"Content-Type": "application/json",
			},
		}
	}

	// hold uploadID
	uploadID := uuid.New()

	// create the version
	versionID := uuid.New()
	if _, err := txn.Exec(
		`insert into public.files_versions (id, files_id, storage_uploads_id) values ($1, $2, $3)`,
		versionID.String(),
		fileID,
		uploadID.String(),
	); err != nil {
		log.Printf("failed to insert version: %v\n", err)
		return handler.HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Body:       `{"error":"failed to create file-version","data":null}`,
			Header: handler.HTTPHeader{
				"Content-Type": "application/json",
			},
		}
	}

	// switch to admin user
	if err := db.SetUser(txn, "admin", uuid.Nil); err != nil {
		return handler.HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Body:       `{"error":"failed to handle authentication (switch-user error)","data":null}`,
			Header: handler.HTTPHeader{
				"Content-Type": "application/json",
			},
		}
	}

	// create the storage_uploads record
	if _, err := txn.Exec(
		`INSERT INTO public.storage_uploads (id, storage_provider_id, file_version_id, provider_id, checksum) VALUES ($1, $2, $3, $4, $5)`,
		uploadID.String(),
		storageProviderID.String(),
		versionID.String(),
		fmt.Sprintf("%s/%s", strings.TrimSuffix(providerID, "/"), strings.Split(hook.Upload.ID, "+")[0]),
		checksum,
	); err != nil {
		log.Printf("failed to insert storage_upload: %v\n", err)
		return handler.HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Body:       `{"error":"failed to create storage-upload record","data":null}`,
			Header: handler.HTTPHeader{
				"Content-Type": "application/json",
			},
		}
	}

	// commit
	if err := txn.Commit(); err != nil {
		log.Printf("failed to commit transaction: %v\n", err)
		return handler.HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Body:       `{"error":"failed to commit changes","data":null}`,
			Header: handler.HTTPHeader{
				"Content-Type": "application/json",
			},
		}
	}

	return handler.HTTPResponse{
		StatusCode: http.StatusNoContent,
		Header: handler.HTTPHeader{
			"Location": fmt.Sprintf("%s:%s", fileID, versionID.String()),
		},
	}
}
