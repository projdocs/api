package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/projdocs/api/config"
	"github.com/projdocs/api/internal/storage/providers"
	"github.com/projdocs/projdocs/packages/go/database"
	"github.com/tus/tusd/v2/pkg/handler"
)

type Provider interface {
	CreateFolder(ctx context.Context, parentID *string, name string, metadata map[string]string) (*string, error)
	ToTusHandler(storageProviderID uuid.UUID, basePath string, parent string) (*handler.Handler, error)
}

func GetProviderFrom(p *database.PublicStorageProvidersSelect) (Provider, error) {
	if !p.IsValid {
		return nil, fmt.Errorf("provider is not valid")
	}

	switch p.Type {
	case "BUILT_IN":
		return providers.NewS3Provider(providers.S3Config{
			Region:          "local",
			Bucket:          "projdocs",
			AccessKeyID:     config.MustGet().S3.AccessKey,
			SecretAccessKey: config.MustGet().S3.SecretKey,
			Endpoint:        fmt.Sprintf("%s/storage/v1/s3", strings.TrimSuffix(config.MustGet().KongURL, "/")),
		})
	case "S3":
		var data struct {
			Bucket      string `json:"bucket"`
			AccessKeyID string `json:"accessKeyId"`
			SecretKey   string `json:"secretKey"`
			Endpoint    string `json:"endpoint"`
			Region      string `json:"region"`
		}

		if err := json.Unmarshal(p.Data.([]byte), &data); err != nil {
			return nil, fmt.Errorf("unmarshal storage provider data: %w", err)
		}

		return providers.NewS3Provider(providers.S3Config{
			Region:          data.Region,
			Bucket:          data.Bucket,
			AccessKeyID:     data.AccessKeyID,
			SecretAccessKey: data.SecretKey,
			Endpoint:        data.Endpoint,
		})
	default:
		return nil, fmt.Errorf("unhandled provider type: %s", p.Type)
	}
}
