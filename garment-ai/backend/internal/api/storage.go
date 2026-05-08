package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"garment-ai/backend/internal/config"
)

type uploadedPhoto struct {
	View      string
	FileName  string
	ObjectKey string
	URL       string
}

func uploadPhotosToMinio(ctx context.Context, cfg config.Config, request UploadPhotosRequest) ([]uploadedPhoto, error) {
	client, err := newMinioClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	if err := ensureBucket(ctx, client, cfg.MinioBucket); err != nil {
		return nil, err
	}

	uploaded := make([]uploadedPhoto, 0, len(request.Photos))
	for index, photo := range request.Photos {
		safeFileName := sanitizeFileName(photo.FileName)
		objectKey := fmt.Sprintf("uploads/%s/%d-%s", request.SessionID, index+1, safeFileName)
		payload, err := decodeBase64Payload(photo.Base64Data)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", photo.FileName, err)
		}

		_, err = client.PutObject(
			ctx,
			cfg.MinioBucket,
			objectKey,
			bytes.NewReader(payload),
			int64(len(payload)),
			minio.PutObjectOptions{ContentType: photo.ContentType},
		)
		if err != nil {
			return nil, fmt.Errorf("upload %s: %w", photo.FileName, err)
		}

		uploaded = append(uploaded, uploadedPhoto{
			View:      photo.View,
			FileName:  safeFileName,
			ObjectKey: objectKey,
			URL:       fmt.Sprintf("http://%s/%s/%s", cfg.MinioEndpoint, cfg.MinioBucket, objectKey),
		})
	}

	return uploaded, nil
}

func newMinioClient(cfg config.Config) (*minio.Client, error) {
	return minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: false,
	})
}

func getObjectFromMinio(ctx context.Context, cfg config.Config, objectKey string) (*minio.Object, minio.ObjectInfo, error) {
	client, err := newMinioClient(cfg)
	if err != nil {
		return nil, minio.ObjectInfo{}, fmt.Errorf("create minio client: %w", err)
	}

	object, err := client.GetObject(ctx, cfg.MinioBucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, minio.ObjectInfo{}, fmt.Errorf("get object %s: %w", objectKey, err)
	}

	info, err := object.Stat()
	if err != nil {
		object.Close()
		return nil, minio.ObjectInfo{}, fmt.Errorf("stat object %s: %w", objectKey, err)
	}

	return object, info, nil
}

func ensureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket %s: %w", bucket, err)
	}

	if exists {
		return nil
	}

	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket %s: %w", bucket, err)
	}

	return nil
}

func decodeBase64Payload(payload string) ([]byte, error) {
	encoded := payload
	if strings.Contains(payload, ",") {
		parts := strings.SplitN(payload, ",", 2)
		encoded = parts[1]
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return data, nil
	}

	return base64.RawStdEncoding.DecodeString(encoded)
}

func sanitizeFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	replacer := strings.NewReplacer(" ", "_", "/", "-", "\\", "-", ":", "-")
	cleaned := replacer.Replace(base)
	if cleaned == "." || cleaned == "" {
		return "photo.jpg"
	}

	return cleaned
}