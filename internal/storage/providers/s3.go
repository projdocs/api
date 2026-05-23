package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

func (p *S3Provider) ToTusHandler(basePath string, uploadPrefix string) (*handler.Handler, error) {

	store := s3store.New(p.bucket, p.client)
	store.ObjectPrefix = uploadPrefix
	composer := handler.NewStoreComposer()
	store.UseIn(composer)

	return handler.NewHandler(handler.Config{
		BasePath:                basePath,
		StoreComposer:           composer,
		RespectForwardedHeaders: true,
		NotifyCompleteUploads:   true,
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
