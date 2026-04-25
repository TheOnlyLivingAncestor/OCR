package storage

import (
	"context"
	"io"
	"log/slog"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage interface {
	Upload(ctx context.Context, data []byte, description string) error
}

type UploadRequest struct {
	File        io.Reader
	Size        int64
	FileName    string
	ContentType string
	Metadata    map[string]string
}

type MinioStorage struct {
	client *minio.Client
	bucket string
	logger *slog.Logger
}

func NewMinioStorage(endpoint string, cred *credentials.Credentials, SSL bool, bucket string, logger *slog.Logger) *MinioStorage {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  cred,
		Secure: SSL,
	})
	logger.Debug("Used data", "endpoint", endpoint, "credentials", cred, "bucket", bucket)
	if err != nil {
		logger.Error("Failed to authorize to MinIO", "error", err)
	}
	logger.Info("Succesfully authenticated to MinIO")
	return &MinioStorage{
		client: minioClient,
		bucket: bucket,
		logger: logger,
	}
}

func (storage *MinioStorage) EnsureBucket(ctx context.Context) error {
	exists, err := storage.client.BucketExists(ctx, storage.bucket)
	if err != nil {
		storage.logger.Error("Error returned while verifying bucket existence", "error", err)
		return err
	}
	if !exists {
		err = storage.client.MakeBucket(ctx, storage.bucket, minio.MakeBucketOptions{})
		if err != nil {
			storage.logger.Error("Failed to create bucket", "name", storage.bucket, "error", err)
			return err
		}
	}
	return nil
}

func (storage *MinioStorage) Upload(ctx context.Context, request UploadRequest) error {
	_, err := storage.client.PutObject(ctx, storage.bucket, request.FileName, request.File, request.Size,
		minio.PutObjectOptions{
			ContentType:  request.ContentType,
			UserMetadata: request.Metadata,
		})
	if err != nil {
		storage.logger.Error("Error occurred during image upload", "error", err)
		return err
	}
	return nil
}
