package storage

import (
	"fmt"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// NewAdapter creates a new storage adapter based on the configuration
func NewAdapter(cfg types.StorageConfig) (Adapter, error) {
	switch cfg.Adapter {
	case "local":
		return NewLocalAdapter(cfg.Local.BasePath)
	case "s3":
		return NewS3Adapter(S3Options{
			Endpoint:        cfg.S3.Endpoint,
			Region:          cfg.S3.Region,
			Bucket:          cfg.S3.Bucket,
			AccessKeyID:     cfg.S3.AccessKeyID,
			SecretAccessKey: cfg.S3.SecretAccessKey,
			UseSSL:          cfg.S3.UseSSL,
		})
	default:
		return nil, fmt.Errorf("unknown storage adapter: %s", cfg.Adapter)
	}
}
