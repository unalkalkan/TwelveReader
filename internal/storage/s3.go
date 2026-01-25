package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Adapter implements the Adapter interface for S3-compatible storage
type S3Adapter struct {
	client *s3.Client
	bucket string
}

// S3Options holds S3 adapter configuration
type S3Options struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
}

// NewS3Adapter creates a new S3 adapter
func NewS3Adapter(opts S3Options) (*S3Adapter, error) {
	ctx := context.Background()

	// Build AWS config
	var cfg aws.Config
	var err error

	if opts.AccessKeyID != "" && opts.SecretAccessKey != "" {
		// Use static credentials
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(opts.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				opts.AccessKeyID,
				opts.SecretAccessKey,
				"",
			)),
		)
	} else {
		// Use default credential chain
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(opts.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with custom endpoint if provided
	var clientOpts []func(*s3.Options)
	if opts.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(opts.Endpoint)
			o.UsePathStyle = true // Required for MinIO and similar services
		})
	}

	client := s3.NewFromConfig(cfg, clientOpts...)

	return &S3Adapter{
		client: client,
		bucket: opts.Bucket,
	}, nil
}

// Put stores data at the given path
func (s *S3Adapter) Put(ctx context.Context, path string, data io.Reader) error {
	// Read all data into memory (for small files this is acceptable)
	// For large files, we'd want to use multipart uploads
	buf, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   bytes.NewReader(buf),
	})

	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}

	return nil
}

// Get retrieves data from the given path
func (s *S3Adapter) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	return result.Body, nil
}

// Delete removes data at the given path
func (s *S3Adapter) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// Exists checks if data exists at the given path
func (s *S3Adapter) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return true, nil
}

// List returns paths matching the given prefix
func (s *S3Adapter) List(ctx context.Context, prefix string) ([]string, error) {
	var paths []string

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key != nil {
				paths = append(paths, *obj.Key)
			}
		}
	}

	return paths, nil
}

// Close cleans up any resources
func (s *S3Adapter) Close() error {
	// No cleanup needed for S3 adapter
	return nil
}
