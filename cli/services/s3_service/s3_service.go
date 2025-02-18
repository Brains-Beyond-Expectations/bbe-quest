package s3_service

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Service struct {
	client *s3.Client
}

func Initialize(ctx context.Context) (*S3Service, error) { // coverage-ignore
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-1"))
	if err != nil {
		return nil, err
	}
	s3Client := s3.NewFromConfig(cfg)
	return &S3Service{client: s3Client}, nil
}

func (s *S3Service) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) { // coverage-ignore
	return s.client.CreateBucket(ctx, params, optFns...)
}

func (s *S3Service) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) { // coverage-ignore
	return s.client.GetObject(ctx, params, optFns...)
}

func (s *S3Service) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) { // coverage-ignore
	return s.client.ListBuckets(ctx, params, optFns...)
}

func (s *S3Service) PutBucketEncryption(ctx context.Context, params *s3.PutBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.PutBucketEncryptionOutput, error) { // coverage-ignore
	return s.client.PutBucketEncryption(ctx, params, optFns...)
}

func (s *S3Service) PutBucketVersioning(ctx context.Context, params *s3.PutBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.PutBucketVersioningOutput, error) { // coverage-ignore
	return s.client.PutBucketVersioning(ctx, params, optFns...)
}

func (s *S3Service) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) { // coverage-ignore
	return s.client.PutObject(ctx, params, optFns...)
}
