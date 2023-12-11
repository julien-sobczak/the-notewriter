package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"storj.io/uplink"
)

var (
	ErrObjectNotExist = errors.New("object does not exist")
)

// Remote provides an abstraction in front of remote implementations.
//
// A remote must be able to save different files:
// - info files (ex: commit-graph)
// - object files (ex: Commit, or File, Note, Flaschard, ...)
// - blob files (ex: medias in various sizes)
//
// A remote is free to save files in any format as long as it can retrieve
// the same field when querying using the same key.
type Remote interface {
	GetObject(key string) ([]byte, error)
	PutObject(key string, content []byte) error
	DeleteObject(key string) error
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

/* Storj */

type StorjRemote struct {
	// Settings
	bucketName string
	// Client
	project *uplink.Project
}

// NewStorjRemoteFromProject instantiates a client using a project (useful for testing purposes).
func NewStorjRemoteFromProject(bucketName string, project *uplink.Project) (*StorjRemote, error) {
	ctx := context.Background()

	// Ensure the desired Bucket within the Project is created.
	_, err := project.EnsureBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("could not ensure bucket: %v", err)
	}

	return &StorjRemote{
		bucketName: bucketName,
		project:    project,
	}, nil
}

// NewStorjRemoteWithCredentials instantiates a client using the access grant.
func NewStorjRemoteWithCredentials(bucketName string, accessGrant string) (*StorjRemote, error) {
	ctx := context.Background()

	// Parse access grant, which contains necessary credentials and permissions.
	access, err := uplink.ParseAccess(accessGrant)
	if err != nil {
		return nil, fmt.Errorf("could not request access grant: %v", err)
	}

	// Open up the Project we will be working with.
	project, err := uplink.OpenProject(ctx, access)
	if err != nil {
		return nil, fmt.Errorf("could not open project: %v", err)
	}

	return NewStorjRemoteFromProject(bucketName, project)
}

func (r *StorjRemote) GetObject(key string) ([]byte, error) {
	ctx := context.Background()

	// Initiate a download of the same object again
	download, err := r.project.DownloadObject(ctx, r.bucketName, key, nil)
	if err != nil {
		return nil, fmt.Errorf("could not open object: %v", err)
	}
	defer download.Close()

	// Read everything from the download stream
	data, err := io.ReadAll(download)
	if err != nil {
		return nil, fmt.Errorf("could not read data: %v", err)
	}

	return data, nil
}

func (r *StorjRemote) PutObject(key string, data []byte) error {
	ctx := context.Background()

	// Initiiate the upload of our Object to the specified bucket and key.
	upload, err := r.project.UploadObject(ctx, r.bucketName, key, &uplink.UploadOptions{
		// No expiration!
	})
	if err != nil {
		return fmt.Errorf("could not initiate upload: %v", err)
	}

	// Copy the data to the upload.
	buf := bytes.NewBuffer(data)
	_, err = io.Copy(upload, buf)
	if err != nil {
		_ = upload.Abort()
		return fmt.Errorf("could not upload data: %v", err)
	}

	// Commit the uploaded object.
	err = upload.Commit()
	if err != nil {
		return fmt.Errorf("could not commit uploaded object: %v", err)
	}

	return nil
}

func (r *StorjRemote) DeleteObject(key string) error {
	ctx := context.Background()
	_, err := r.project.DeleteObject(ctx, r.bucketName, key)
	if err != nil {
		return err
	}
	return nil
}
