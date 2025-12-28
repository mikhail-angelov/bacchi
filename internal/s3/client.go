// Package s3 provides a wrapper around the AWS S3 SDK.
package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client is a wrapper around the AWS S3 client.
type Client struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewClient creates a new S3 client with the given credentials and configuration.
func NewClient(ctx context.Context, bucket, region, endpoint, accessKey, secretKey, prefix string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load SDK config: %w", err)
	}

	return &Client{
		client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
			if endpoint != "" {
				o.BaseEndpoint = aws.String(endpoint)
			}
		}),
		bucket: bucket,
		prefix: prefix,
	}, nil
}

// UploadFile uploads a local file to S3.
func (c *Client) UploadFile(ctx context.Context, filePath string) error {
	file, err := os.Open(filePath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	key := filepath.Join(c.prefix, filepath.Base(filePath))

	uploader := manager.NewUploader(c.client)
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// ListBackups lists all backup files in the S3 bucket under the specified prefix.
func (c *Client) ListBackups(ctx context.Context) ([]string, error) {
	var backups []string
	paginator := s3.NewListObjectsV2Paginator(c.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(c.prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list S3 objects: %w", err)
		}

		for _, obj := range page.Contents {
			backups = append(backups, *obj.Key)
		}
	}

	return backups, nil
}

// DeleteFile deletes a file from S3 by its key.
func (c *Client) DeleteFile(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete S3 object: %w", err)
	}
	return nil
}

// DownloadFile downloads a file from S3 to a local target path.
func (c *Client) DownloadFile(ctx context.Context, key, targetPath string) error {
	file, err := os.Create(targetPath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	downloader := manager.NewDownloader(c.client)
	_, err = downloader.Download(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download from S3: %w", err)
	}

	return nil
}
