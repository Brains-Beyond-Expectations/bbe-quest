package mocks

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/mock"
)

type MockS3Service struct {
	mock.Mock
}

func (mock *MockS3Service) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	args := mock.Called(ctx, params, optFns)

	return args.Get(0).(*s3.CreateBucketOutput), args.Error(1)
}

func (mock *MockS3Service) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := mock.Called(ctx, params, optFns)

	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (mock *MockS3Service) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	args := mock.Called(ctx, params, optFns)

	return args.Get(0).(*s3.ListBucketsOutput), args.Error(1)
}

func (mock *MockS3Service) PutBucketEncryption(ctx context.Context, params *s3.PutBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.PutBucketEncryptionOutput, error) {
	args := mock.Called(ctx, params, optFns)

	return args.Get(0).(*s3.PutBucketEncryptionOutput), args.Error(1)
}

func (mock *MockS3Service) PutBucketVersioning(ctx context.Context, params *s3.PutBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.PutBucketVersioningOutput, error) {
	args := mock.Called(ctx, params, optFns)

	return args.Get(0).(*s3.PutBucketVersioningOutput), args.Error(1)
}

func (mock *MockS3Service) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := mock.Called(ctx, params, optFns)

	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}
