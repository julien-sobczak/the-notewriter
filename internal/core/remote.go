package core

import (
	"fmt"
	"os"
)

type Remote interface {
	ListObjects() ([]*Object, error)
	GetObject(oid string) (*Object, error)
	PutObject(oid string, obj *Object) (*Object, error)
	DeleteObject(oid string) error
	// Note: File permissions are not important concerning object. MTime, etc. must be stored inside the object definitions if useful.
}

// Typical files:
//   ob/jectid // YAML/JSON single object
//   bl/ob // Binary single blob
//   index // objectid => YAML/JSON path to commit file
//   commit-graph // YAML/JSON list of commits id
// Note: don't use extension to match with Git standard namings

/* FS */

type FSRemote struct {
	path string
	// Use classic FS APIs to satisfy interface
}

func NewFSRemote(dirpath string) (*FSRemote, error) {
	stat, err := os.Stat("/path/to/whatever")
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

func (r *FSRemote) ListObjects() ([]*Object, error) {
	// TODO
	return nil, nil
}

func (r *FSRemote) GetObject(oid string) (*Object, error) {
	// TODO
	return nil, nil
}

func (r *FSRemote) PutObject(oid string, obj *Object) (*Object, error) {
	// TODO
	return nil, nil
}

func (r *FSRemote) DeleteObject(oid string) error {
	// TODO
	return nil
}

/* S3 */

type S3Remote struct {
	accessKey  string
	secretKey  string
	bucketName string
	// Use S3 API to satisfy interface
}

func NewS3RemoteFromCredentials(bucketName string, accessKey, secretKey string) (*S3Remote, error) {
	return &S3Remote{
		accessKey:  accessKey,
		secretKey:  secretKey,
		bucketName: bucketName,
	}, nil
}

func (r *S3Remote) ListObjects() ([]*Object, error) {
	// TODO
	return nil, nil
}

func (r *S3Remote) GetObject(oid string) (*Object, error) {
	// TODO
	return nil, nil
}

func (r *S3Remote) PutObject(oid string, obj *Object) (*Object, error) {
	// TODO
	return nil, nil
}

func (r *S3Remote) DeleteObject(oid string) error {
	// TODO
	return nil
}
