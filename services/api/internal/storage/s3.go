package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Store interface {
	PutDataURL(ctx context.Context, evidenceID string, dataURL string) (string, error)
	ReadDataURL(ctx context.Context, key string) (string, string, error)
}

type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(ctx context.Context, endpoint, accessKey, secretKey, bucket string, useSSL bool) (S3, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return S3{}, err
	}

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return S3{}, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return S3{}, err
		}
	}

	return S3{client: client, bucket: bucket}, nil
}

func (s S3) PutDataURL(ctx context.Context, evidenceID string, dataURL string) (string, error) {
	mediaType, body, err := splitDataURL(dataURL)
	if err != nil {
		return "", err
	}

	key := evidenceID + extForMediaType(mediaType)
	_, err = s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(body), int64(len(body)), minio.PutObjectOptions{
		ContentType: mediaType,
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s S3) ReadDataURL(ctx context.Context, key string) (string, string, error) {
	if key == "" {
		return "", "", errors.New("invalid object key")
	}

	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return "", "", err
	}
	defer object.Close()

	body, err := io.ReadAll(object)
	if err != nil {
		return "", "", err
	}

	mediaType := mime.TypeByExtension(filepath.Ext(key))
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	return mediaType, "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(body), nil
}

func extForMediaType(mediaType string) string {
	if mediaType == "image/png" {
		return ".png"
	}
	exts, _ := mime.ExtensionsByType(mediaType)
	if len(exts) > 0 {
		return exts[0]
	}
	return ".bin"
}

func splitDataURL(dataURL string) (string, []byte, error) {
	if !strings.HasPrefix(dataURL, "data:") {
		return "", nil, errors.New("expected data url")
	}
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return "", nil, errors.New("invalid data url")
	}
	meta := strings.TrimPrefix(parts[0], "data:")
	if !strings.HasSuffix(meta, ";base64") {
		return "", nil, errors.New("expected base64 data url")
	}
	mediaType := strings.TrimSuffix(meta, ";base64")
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	body, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, fmt.Errorf("decode data url: %w", err)
	}
	return mediaType, body, nil
}
