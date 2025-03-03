package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	ErrObjectNotExist = errors.New("object does not exist")
)

// Remote provides an abstraction in front of remote implementations.
//
// A remote must be able to save different files:
// - info files (ex: index)
// - pack files (ex: files, medias)
// - blob files (ex: medias in various sizes)
//
// A remote is free to save files in any format as long as it can retrieve
// the same field when querying using the same key.
type Remote interface {
	GetObject(key string) ([]byte, error)
	PutObject(key string, content []byte) error
	DeleteObject(key string) error
	GC() error
	// Note: File permissions are not important concerning object. MTime, etc. must be stored inside the object definitions if useful.
}

/* FS */

type FSRemote struct {
	path string
	// Use classic FS APIs to satisfy interface
}

func NewFSRemote(dirpath string) (*FSRemote, error) {
	stat, err := os.Stat(dirpath)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dirpath)
	}

	return &FSRemote{
		path: dirpath,
	}, nil
}

func (r *FSRemote) GetObject(key string) ([]byte, error) {
	path := filepath.Join(r.path, key)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrObjectNotExist
	}
	return data, err
}

func (r *FSRemote) PutObject(key string, data []byte) error {
	dirPath := filepath.Join(r.path, filepath.Dir(key))
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return err
	}
	filePath := filepath.Join(r.path, key)
	return os.WriteFile(filePath, data, 0644)
}

func (r *FSRemote) DeleteObject(key string) error {
	path := filepath.Join(r.path, key)
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrObjectNotExist
	}
	return os.Remove(path)
}

func (r *FSRemote) GC() error {
	// Not implemented
	return nil
}

/* S3 */

type S3Remote struct {
	// Settings
	endpoint   string
	accessKey  string
	secretKey  string
	bucketName string
	// Client
	minioClient *minio.Client
}

func NewS3RemoteWithCredentials(endpoint string, bucketName string, accessKey, secretKey string, secure bool) (*S3Remote, error) {
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}

	return &S3Remote{
		endpoint:    endpoint,
		accessKey:   accessKey,
		secretKey:   secretKey,
		bucketName:  bucketName,
		minioClient: minioClient,
	}, nil
}

func (r *S3Remote) GetObject(key string) ([]byte, error) {
	object, err := r.minioClient.GetObject(context.Background(), r.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	stat, err := object.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size == 0 {
		return nil, ErrObjectNotExist
	}
	defer object.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(object)
	return buf.Bytes(), nil
}

func (r *S3Remote) PutObject(key string, data []byte) error {
	_, err := r.minioClient.PutObject(context.Background(), r.bucketName, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	return err
}

func (r *S3Remote) DeleteObject(key string) error {
	_, err := r.GetObject(key)
	if err != nil {
		return err
	}
	return r.minioClient.RemoveObject(context.Background(), r.bucketName, key, minio.RemoveObjectOptions{})
}

func (r *S3Remote) GC() error {
	// Not implemented
	return nil
}
