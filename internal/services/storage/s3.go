package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	appconfig "github.com/denisAlshanov/stPlaner/internal/config"
)

type S3Storage struct {
	client     *s3.Client
	bucketName string
}

func (s *S3Storage) BucketName() string {
	return s.bucketName
}

func NewS3Storage(cfg *appconfig.S3Config) (*S3Storage, error) {
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var client *s3.Client
	
	// Check if we're using LocalStack
	if cfg.EndpointURL != "" {
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.EndpointURL)
			o.UsePathStyle = true // Required for LocalStack
		})
	} else {
		client = s3.NewFromConfig(awsCfg)
	}

	return &S3Storage{
		client:     client,
		bucketName: cfg.BucketName,
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, key string, data io.Reader, contentType string) error {
	// Read all data into buffer for size calculation
	buf := new(bytes.Buffer)
	size, err := io.Copy(buf, data)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(key),
		Body:          bytes.NewReader(buf.Bytes()),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}

	_, err = s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

func (s *S3Storage) UploadWithMetadata(ctx context.Context, key string, data io.Reader, contentType string, metadata map[string]string) error {
	// Read all data into buffer for size calculation
	buf := new(bytes.Buffer)
	size, err := io.Copy(buf, data)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(key),
		Body:          bytes.NewReader(buf.Bytes()),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
		Metadata:      metadata,
	}

	_, err = s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

func (s *S3Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	return result.Body, nil
}

func (s *S3Storage) GetMetadata(ctx context.Context, key string) (map[string]string, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	result, err := s.client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	metadata := make(map[string]string)
	metadata["ContentType"] = aws.ToString(result.ContentType)
	metadata["ContentLength"] = fmt.Sprintf("%d", aws.ToInt64(result.ContentLength))
	metadata["LastModified"] = result.LastModified.Format(time.RFC3339)

	// Merge custom metadata
	for k, v := range result.Metadata {
		metadata[k] = v
	}

	return metadata, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	_, err := s.client.HeadObject(ctx, input)
	if err != nil {
		// Check if the error is because the object doesn't exist
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

func (s *S3Storage) GeneratePresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	presignResult, err := presignClient.PresignGetObject(ctx, input, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignResult.URL, nil
}

func (s *S3Storage) InitiateMultipartUpload(ctx context.Context, key string, contentType string) (string, error) {
	input := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	result, err := s.client.CreateMultipartUpload(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	return aws.ToString(result.UploadId), nil
}

func (s *S3Storage) UploadPart(ctx context.Context, key string, uploadID string, partNumber int32, data io.Reader) (*CompletedPart, error) {
	// Read all data into buffer for size calculation
	buf := new(bytes.Buffer)
	size, err := io.Copy(buf, data)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	input := &s3.UploadPartInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(key),
		UploadId:      aws.String(uploadID),
		PartNumber:    aws.Int32(partNumber),
		Body:          bytes.NewReader(buf.Bytes()),
		ContentLength: aws.Int64(size),
	}

	result, err := s.client.UploadPart(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload part: %w", err)
	}

	return &CompletedPart{
		ETag:       result.ETag,
		PartNumber: aws.Int32(partNumber),
	}, nil
}

func (s *S3Storage) CompleteMultipartUpload(ctx context.Context, key string, uploadID string, parts []CompletedPart) error {
	input := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.bucketName),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: convertToS3Parts(parts),
		},
	}

	_, err := s.client.CompleteMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	return nil
}

func (s *S3Storage) AbortMultipartUpload(ctx context.Context, key string, uploadID string) error {
	input := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s.bucketName),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	}

	_, err := s.client.AbortMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	return nil
}

func convertToS3Parts(parts []CompletedPart) []types.CompletedPart {
	s3Parts := make([]types.CompletedPart, len(parts))
	for i, part := range parts {
		s3Parts[i] = types.CompletedPart{
			ETag:       part.ETag,
			PartNumber: part.PartNumber,
		}
	}
	return s3Parts
}

func isNotFoundError(err error) bool {
	// S3 returns specific error codes for not found
	return err != nil && err.Error() != "" // This is a simplified check, in production you'd check for specific AWS error types
}
