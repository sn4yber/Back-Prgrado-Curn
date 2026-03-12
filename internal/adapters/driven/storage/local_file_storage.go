package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type LocalFileStorage struct {
	baseDir string
	baseURL string
}

func NewLocalFileStorage(baseDir, baseURL string) *LocalFileStorage {
	return &LocalFileStorage{baseDir: baseDir, baseURL: baseURL}
}

func (s *LocalFileStorage) Save(_ context.Context, objectKey string, _ string, data []byte) (string, error) {
	targetPath := filepath.Join(s.baseDir, objectKey)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", fmt.Errorf("localFileStorage.Save mkdir: %w", err)
	}

	if err := os.WriteFile(targetPath, data, 0o644); err != nil {
		return "", fmt.Errorf("localFileStorage.Save write: %w", err)
	}

	return fmt.Sprintf("%s/%s", s.baseURL, filepath.ToSlash(objectKey)), nil
}
