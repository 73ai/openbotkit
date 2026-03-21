package backup

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type R2Backend struct {
	client *minio.Client
	bucket string
}

func NewR2Backend(endpoint, accessKey, secretKey, bucket string) (*R2Backend, error) {
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: true,
		Region: "auto",
	})
	if err != nil {
		return nil, fmt.Errorf("create R2 client: %w", err)
	}

	return &R2Backend{client: client, bucket: bucket}, nil
}

func (b *R2Backend) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	_, err := b.client.PutObject(ctx, b.bucket, key, r, size, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}

func (b *R2Backend) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := b.client.GetObject(ctx, b.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", key, err)
	}
	return obj, nil
}

func (b *R2Backend) Head(ctx context.Context, key string) (bool, error) {
	_, err := b.client.StatObject(ctx, b.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		resp := minio.ToErrorResponse(err)
		if resp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("head %s: %w", key, err)
	}
	return true, nil
}

func (b *R2Backend) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	for obj := range b.client.ListObjects(ctx, b.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("list %s: %w", prefix, obj.Err)
		}
		keys = append(keys, obj.Key)
	}
	return keys, nil
}

func (b *R2Backend) Delete(ctx context.Context, key string) error {
	if err := b.client.RemoveObject(ctx, b.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}

func ValidateR2(ctx context.Context, endpoint, accessKey, secretKey, bucket string) error {
	backend, err := NewR2Backend(endpoint, accessKey, secretKey, bucket)
	if err != nil {
		return err
	}
	exists, err := backend.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist", bucket)
	}
	return nil
}

var _ Backend = (*R2Backend)(nil)
