package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3 supports S3-compatible object solutions. Ex:
// - CloudFlare R2 (10GB free tier but credit card required)
// - Filebase (5GB free without credit card)
type S3 struct {
	// Settings
	endpoint   string
	accessKey  string
	secretKey  string
	bucketName string
	// Client
	minioClient *minio.Client
	// See https://docs.filebase.com/code-development-+-sdks/code-development/aws-sdk-go-golang for Filebase implementation using AWS SDK
}

func NewS3WithCredentials(endpoint string, bucketName string, accessKey, secretKey string, secure bool) (*S3, error) {
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}

	return &S3{
		endpoint:    endpoint,
		accessKey:   accessKey,
		secretKey:   secretKey,
		bucketName:  bucketName,
		minioClient: minioClient,
	}, nil
}

func (r *S3) GetObject(key string) ([]byte, error) {
	object, err := r.minioClient.GetObject(context.Background(), r.bucketName, key, minio.GetObjectOptions{})
	var minioErr minio.ErrorResponse
	if err != nil && errors.As(err, &minioErr) && minioErr.Code == "NoSuchKey" {
		return nil, ErrObjectNotExist
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get object %q: %w", key, err)
	}
	stat, err := object.Stat()
	if err != nil && errors.As(err, &minioErr) && minioErr.Code == "NoSuchKey" {
		return nil, ErrObjectNotExist
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat object %q: %w", key, err)
	}
	if stat.Size == 0 {
		return nil, ErrObjectNotExist
	}
	defer object.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(object)
	return buf.Bytes(), nil
}

func (r *S3) PutObject(key string, data []byte) error {
	_, err := r.minioClient.PutObject(context.Background(), r.bucketName, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	return err
}

func (r *S3) DeleteObject(key string) error {
	if _, err := r.GetObject(key); err != nil {
		return err
	}
	return r.minioClient.RemoveObject(context.Background(), r.bucketName, key, minio.RemoveObjectOptions{})
}

func (r *S3) GC() error {
	// Not implemented
	return nil
}
