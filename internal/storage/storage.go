// Package storage saves files to an S3-compatible bucket, like Cloudflare R2.
package storage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config holds the bucket's address and credentials.
type Config struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

type S3 struct {
	client *s3.Client
	bucket string
}

func NewS3(cfg Config) *S3 {
	client := s3.New(s3.Options{
		Region:       cfg.Region,
		BaseEndpoint: aws.String(cfg.Endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
	})
	return &S3{client: client, bucket: cfg.Bucket}
}

// Put saves body under key, with its content type and user metadata.
func (s *S3) Put(ctx context.Context, key string, body []byte, contentType string, metadata map[string]string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
		Metadata:    metadata,
	})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}
