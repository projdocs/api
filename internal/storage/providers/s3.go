package providers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/projdocs/api/internal/db"
	"github.com/projdocs/api/internal/router/middleware"
	"github.com/tus/tusd/v2/pkg/handler"
	"github.com/tus/tusd/v2/pkg/s3store"
)

// S3Config carries the per-request S3 credentials and target.
// Your Majesty should populate this from whatever source is appropriate
// (database, JWT claims, request headers, etc.)
type S3Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // optional; leave empty for real AWS S3
}

type S3Provider struct {
	client *s3.Client
	bucket string
}

func (p *S3Provider) ToTusHandler(storageProviderID uuid.UUID, basePath string, uploadPrefix string) (*handler.Handler, error) {

	store := s3store.New(p.bucket, p.client)
	store.ObjectPrefix = fmt.Sprintf("%s/", strings.TrimPrefix(strings.TrimSuffix(uploadPrefix, "/"), "/"))
	composer := handler.NewStoreComposer()
	store.UseIn(composer)

	return handler.NewHandler(handler.Config{
		BasePath:                basePath,
		StoreComposer:           composer,
		RespectForwardedHeaders: true,
		NotifyCompleteUploads:   true,
		PreFinishResponseCallback: func(hook handler.HookEvent) (handler.HTTPResponse, error) {

			folderID := strings.Split(hook.HTTPRequest.URI, "/")[5]
			providerID := fmt.Sprintf("%s/%s", strings.TrimPrefix(strings.TrimSuffix(uploadPrefix, "/"), "/"), strings.Split(hook.Upload.ID, "+")[0])

			// get db connection
			var pg *sql.DB
			if _pg, err := db.Get(); err != nil {
				return handler.HTTPResponse{
					StatusCode: http.StatusBadRequest,
					Body:       `{"error":"unable to connect to database","data":null}`,
					Header: handler.HTTPHeader{
						"Content-Type": "application/json",
					},
				}, nil
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
				}, nil
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
				}, nil
			}

			// hold uploadID
			uploadID := uuid.New()

			// create the file
			fileID := uuid.New()
			fileName := "new-file"
			if _fileName, ok := hook.Upload.MetaData["filename"]; ok && _fileName != "" {
				fileName = _fileName
			}
			if _, err := txn.Exec(
				`insert into public.files (id, folder_id, name) values ($1, $2, $3)`,
				fileID.String(),
				folderID,
				fileName,
			); err != nil {
				return handler.HTTPResponse{
					StatusCode: http.StatusBadRequest,
					Body:       `{"error":"failed to create file","data":null}`,
					Header: handler.HTTPHeader{
						"Content-Type": "application/json",
					},
				}, nil
			}

			// create the version
			versionID := uuid.New()
			if _, err := txn.Exec(
				`insert into public.files_versions (id, files_id, storage_uploads_id) values ($1, $2, $3)`,
				versionID.String(),
				fileID.String(),
				uploadID.String(),
			); err != nil {
				log.Printf("failed to insert version: %v\n", err)
				return handler.HTTPResponse{
					StatusCode: http.StatusBadRequest,
					Body:       `{"error":"failed to create file-version","data":null}`,
					Header: handler.HTTPHeader{
						"Content-Type": "application/json",
					},
				}, nil
			}

			head, err := p.client.HeadObject(context.Background(), &s3.HeadObjectInput{
				Bucket: aws.String(p.bucket),
				Key:    aws.String(providerID),
			})
			if err != nil {
				return handler.HTTPResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       `{"error":"failed to retrieve object metadata","data":null}`,
					Header:     handler.HTTPHeader{"Content-Type": "application/json"},
				}, nil
			}

			checksum := strings.Trim(aws.ToString(head.ETag), `"`)

			if err := db.SetUser(txn, "admin", uuid.Nil); err != nil {
				return handler.HTTPResponse{
					StatusCode: http.StatusBadRequest,
					Body:       `{"error":"failed to handle authentication (switch-user error)","data":null}`,
					Header: handler.HTTPHeader{
						"Content-Type": "application/json",
					},
				}, nil
			}

			// create the storage_uploads record
			if _, err := txn.Exec(
				`INSERT INTO public.storage_uploads (id, storage_provider_id, file_version_id, provider_id, checksum) VALUES ($1, $2, $3, $4, $5)`,
				uploadID.String(),
				storageProviderID.String(),
				versionID.String(),
				providerID,
				checksum,
			); err != nil {
				log.Printf("failed to insert storage_upload: %v\n", err)
				return handler.HTTPResponse{
					StatusCode: http.StatusBadRequest,
					Body:       `{"error":"failed to create storage-upload record","data":null}`,
					Header: handler.HTTPHeader{
						"Content-Type": "application/json",
					},
				}, nil
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
				}, nil
			}

			return handler.HTTPResponse{
				StatusCode: http.StatusNoContent,
				Header: handler.HTTPHeader{
					"Location": fmt.Sprintf("%s:%s", fileID.String(), versionID.String()),
				},
			}, nil
		},
	})
}

func NewS3Provider(cfg S3Config) (*S3Provider, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("aws config error: %w", err)
	}

	return &S3Provider{
		bucket: cfg.Bucket,
		client: s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			if cfg.Endpoint != "" {
				o.BaseEndpoint = &cfg.Endpoint
				o.UsePathStyle = true
			}
		}),
	}, nil
}

func (p *S3Provider) CreateFolder(ctx context.Context, parent *string, name string, metadata map[string]string) (*string, error) {

	path := ""
	if parent != nil {
		path = *parent
	}

	key := fmt.Sprintf(
		"%s/%s/",
		strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/"),
		strings.TrimSuffix(strings.TrimPrefix(name, "/"), "/"))

	key = strings.TrimPrefix(key, "/") // when parent is empty
	metaKey := key + ".metadata"

	metaBody, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("metadata json marshal error: %w", err)
	}

	// create the folder
	if _, err := p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &p.bucket,
		Key:           &key,
		Body:          bytes.NewReader([]byte{}),
		ContentLength: aws.Int64(0),
		Metadata:      metadata,
	}); err != nil {
		return nil, fmt.Errorf("create folder error: %w", err)
	}

	// create the metadata file
	if _, err := p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &p.bucket,
		Key:           &metaKey,
		Body:          bytes.NewReader(metaBody),
		ContentLength: aws.Int64(int64(len(metaBody))),
		ContentType:   aws.String("application/json"),
	}); err != nil {
		return nil, fmt.Errorf("create metadata error: %w", err)
	}

	return &key, nil
}
