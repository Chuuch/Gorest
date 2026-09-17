package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/chuuch/gorest/internal/config"
)

type S3Store struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
	ttl     time.Duration
}

func newClient(endpoint, region, accessKey, secretKey string, usePathStyle bool) *s3.Client {
	cfg := aws.Config{
		Region: region,
		Credentials: aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = usePathStyle
	})
}

func NewS3Store(cfg config.StorageConfig) (*S3Store, error) {
	opClient := newClient(
		cfg.Endpoint,
		cfg.Region,
		cfg.AccessKey,
		cfg.SecretKey,
		cfg.UsePathStyle,
	)
	signClient := newClient(
		cfg.PublicEndpoint,
		cfg.Region,
		cfg.AccessKey,
		cfg.SecretKey,
		cfg.UsePathStyle,
	)

	return &S3Store{
		client:  opClient,
		presign: s3.NewPresignClient(signClient),
		bucket:  cfg.Bucket,
		ttl:     cfg.PresignTTL,
	}, nil
}

func (s *S3Store) EnsureBucket(ctx context.Context, allowedOrigins []string) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		_, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(s.bucket),
		})
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
	}

	if len(allowedOrigins) == 0 {
		return nil
	}
	_, err = s.client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(s.bucket),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{
				{
					AllowedHeaders: []string{"*"},
					AllowedMethods: []string{"GET", "PUT", "HEAD"},
					AllowedOrigins: allowedOrigins,
					ExposeHeaders:  []string{"ETag", "Content-Length", "Content-Type"},
					MaxAgeSeconds:  aws.Int32(3600),
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("put bucket cors: %w", err)
	}

	return nil
}

func (s *S3Store) PresignPut(
	ctx context.Context,
	key, contentType string,
) (string, error) {
	out, err := s.presign.PresignPutObject(
		ctx,
		&s3.PutObjectInput{
			Bucket:      aws.String(s.bucket),
			Key:         aws.String(key),
			ContentType: aws.String(contentType),
		},
		s3.WithPresignExpires(s.ttl),
	)
	if err != nil {
		return "", fmt.Errorf("presign put: %w", err)
	}

	return out.URL, nil
}

func (s *S3Store) PresignGet(
	ctx context.Context,
	key, filename string,
) (string, error) {
	out, err := s.presign.PresignGetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket:                     aws.String(s.bucket),
			Key:                        aws.String(key),
			ResponseContentDisposition: aws.String(contentDisposition(filename)),
		},
		s3.WithPresignExpires(s.ttl),
	)
	if err != nil {
		return "", fmt.Errorf("presign get: %w", err)
	}

	return out.URL, nil
}

func contentDisposition(filename string) string {
	safe := strings.ReplaceAll(filename, `"`, "'")
	return `attachment; filename="` + safe + `"`
}
