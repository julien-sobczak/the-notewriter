//go:build integration

package core

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestS3Remote(t *testing.T) {
	r, _ := SetUpS3Remote(t)

	// Add a file
	err := r.PutObject("index", []byte(`
committed_at: 2023-01-01T01:14:30Z
`))
	require.NoError(t, err)

	// Read the wrong file
	_, err = r.GetObject("info/index")
	require.Error(t, err)

	// Read the correct file
	data, err := r.GetObject("index")
	require.NoError(t, err)
	require.Equal(t, []byte(`
committed_at: 2023-01-01T01:14:30Z
`), data)

	// Update the file
	r.PutObject("index", []byte(`
committed_at: 2023-11-11T11:14:30Z
`))
	// Reread the file
	data, err = r.GetObject("index")
	require.NoError(t, err)
	require.Equal(t, []byte(`
committed_at: 2023-11-11T11:14:30Z
`), data)

	// Delete the file
	err = r.DeleteObject("index")
	require.NoError(t, err)

	// Delete a missing file
	err = r.DeleteObject("index")
	require.Error(t, err)
}

/* Test Helpers */

func SetUpS3Remote(t *testing.T) (*S3Remote, *minio.Client) {
	// Settings
	accessKey := "XXX"      // at least 3 characters
	secretKey := "XXXXXXXX" // at least 8 characters
	bucketName := "my-bucket"

	ctx := context.Background()

	// Start the container
	// (See documentation https://golang.testcontainers.org/quickstart/)
	// (See example https://github.com/romnn/testcontainers/blob/v0.2.0/examples/minio/minio_example.go)
	req := testcontainers.ContainerRequest{
		Image: "minio/minio:RELEASE.2023-02-27T18-10-45Z", // Check https://hub.docker.com/r/minio/minio/tags
		Env: map[string]string{
			"MINIO_ACCESS_KEY": accessKey,
			"MINIO_SECRET_KEY": secretKey,
		},
		Cmd:          []string{"server", "/data"},
		ExposedPorts: []string{"9000"},
		WaitingFor:   wait.ForLog("MinIO Object Storage Server").WithStartupTimeout(10 * time.Second),
	}
	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Error(err)
	}
	t.Cleanup(func() {
		if err := minioContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err.Error())
		}
	})

	// Extract endpoint
	host, err := minioContainer.Host(ctx)
	require.NoError(t, err)
	minioPort, err := nat.NewPort("", "9000")
	require.NoError(t, err)
	port, err := minioContainer.MappedPort(ctx, minioPort)
	require.NoError(t, err)
	endpoint := fmt.Sprintf("%s:%s", host, port.Port())

	// Create the S3 remote
	remoteClient, err := NewS3RemoteWithCredentials(endpoint, bucketName, accessKey, secretKey, false)
	require.NoError(t, err)

	// Create the Minio client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	require.NoError(t, err)

	// Create the bucket
	if err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
		// Check if the bucket already exists
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if !(errBucketExists == nil && exists) {
			log.Fatalf("failed to create bucket %q: %v", bucketName, err)
		}
	}

	return remoteClient, minioClient
}
