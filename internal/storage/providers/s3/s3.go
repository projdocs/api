package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/projdocs/api/internal/storage/types"
	"github.com/tus/tusd/v2/pkg/handler"
	"github.com/tus/tusd/v2/pkg/s3store"
)

// Config carries the per-request S3 credentials and target.
// Your Majesty should populate this from whatever source is appropriate
// (database, JWT claims, request headers, etc.)
type Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // optional; leave empty for real AWS S3
}

type Provider struct {
	client *s3.Client
	bucket string
}

func NewProvider(cfg Config) (*Provider, error) {
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

	return &Provider{
		bucket: cfg.Bucket,
		client: s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			if cfg.Endpoint != "" {
				o.BaseEndpoint = &cfg.Endpoint
				o.UsePathStyle = true
			}
		}),
	}, nil
}

func (p *Provider) ToTusHandler(
	storageProviderId uuid.UUID,
	basePath string,
	uploadPrefix string,
	callback types.Callback,
) (*handler.Handler, error) {

	store := s3store.New(p.bucket, p.client)
	store.ObjectPrefix = fmt.Sprintf("%s/", strings.TrimPrefix(strings.TrimSuffix(uploadPrefix, "/"), "/"))
	composer := handler.NewStoreComposer()
	store.UseIn(composer)

	return handler.NewHandler(handler.Config{
		BasePath:                basePath,
		StoreComposer:           composer,
		RespectForwardedHeaders: true,
		NotifyCompleteUploads:   true,
		PreFinishResponseCallback: func(hook handler.HookEvent) (response handler.HTTPResponse, _ error) {

			// catch errors
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic: %v", r)
					response = handler.HTTPResponse{
						StatusCode: http.StatusInternalServerError,
						Body:       `{"error":"an unexpected error occurred","data":null}`,
						Header:     handler.HTTPHeader{"Content-Type": "application/json"},
					}
				}
			}()

			providerID := fmt.Sprintf("%s/%s", strings.TrimPrefix(strings.TrimSuffix(uploadPrefix, "/"), "/"), strings.Split(hook.Upload.ID, "+")[0])
			head, err := p.client.HeadObject(context.Background(), &s3.HeadObjectInput{
				Bucket: aws.String(p.bucket),
				Key:    aws.String(providerID),
			})
			if err != nil {
				response = handler.HTTPResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       `{"error":"failed to retrieve object metadata","data":null}`,
					Header:     handler.HTTPHeader{"Content-Type": "application/json"},
				}
				return
			}

			checksum := strings.Trim(aws.ToString(head.ETag), `"`)
			response = callback(
				fmt.Sprintf("%s/%s", strings.TrimSuffix(providerID, "/"), strings.Split(hook.Upload.ID, "+")[0]),
				storageProviderId,
				basePath,
				uploadPrefix,
				checksum,
				hook,
			)
			return
		},
	})
}

func (p *Provider) CreateFolder(ctx context.Context, parent *string, name string, metadata map[string]string) (*string, error) {

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

func (p *Provider) GetContent(
	ctx context.Context,
	id string,
	start int64,
	end int64,
) ([]byte, error) {

	result, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(id),
		Range:  aws.String(fmt.Sprintf("bytes=%d-%d", start, end)),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get object: %w", err)
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}
