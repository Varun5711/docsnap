package storage

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

type Store interface {
	PutDataURL(ctx context.Context, evidenceID string, dataURL string) (string, error)
	ReadDataURL(ctx context.Context, key string) (string, string, error)
}

type Local struct {
	root string
}

func NewLocal(root string) Local {
	return Local{root: root}
}

func (l Local) PutDataURL(ctx context.Context, evidenceID string, dataURL string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	mediaType, body, err := splitDataURL(dataURL)
	if err != nil {
		return "", err
	}
	exts, _ := mime.ExtensionsByType(mediaType)
	ext := ".bin"
	if len(exts) > 0 {
		ext = exts[0]
	}
	if mediaType == "image/png" {
		ext = ".png"
	}

	key := evidenceID + ext
	path := filepath.Join(l.root, key)
	if err := os.MkdirAll(l.root, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return key, nil
}

func (l Local) ReadDataURL(ctx context.Context, key string) (string, string, error) {
	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	default:
	}

	if key == "" || strings.Contains(key, "..") || strings.ContainsRune(key, filepath.Separator) {
		return "", "", errors.New("invalid object key")
	}

	path := filepath.Join(l.root, key)
	body, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	mediaType := mime.TypeByExtension(filepath.Ext(key))
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	return mediaType, "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(body), nil
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
