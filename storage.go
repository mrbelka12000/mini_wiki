package mini_wiki

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type (
	Storage struct {
		client *minio.Client
		addr   string
		bucket string
	}
)

func GetStorage(cfg Config) (*Storage, error) {

	minioClient, err := minio.New(cfg.MinIOAddr, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %v", err)
	}

	return &Storage{
		client: minioClient,
		bucket: cfg.MinIOBucket,
		addr:   cfg.MinIOAddr,
	}, nil
}

func (s *Storage) UploadFile(ctx context.Context, file io.Reader, objectName, contentType string, fileSize int64) (string, error) {
	info, err := s.client.PutObject(ctx, s.bucket, objectName, file, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload file: %v", err)
	}

	return info.Key, nil
}

func (s *Storage) DownloadFile(ctx context.Context, w http.ResponseWriter, objectName string) error {
	object, err := s.client.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("download file: %v", err)
	}

	objectInfo, err := object.Stat()
	if err != nil {
		return fmt.Errorf("get stats: %v", err)
	}

	// Set HTTP headers to force the browser to download the file
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", objectName))
	w.Header().Set("Content-Type", objectInfo.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", objectInfo.Size))

	// Stream the file from MinIO to the HTTP response using io.Copy
	if _, err := io.Copy(w, object); err != nil {
		return fmt.Errorf("copy: %v", err)
	}

	return nil
}
