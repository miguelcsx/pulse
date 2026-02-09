package store

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Storage defines the interface for file storage operations.
type Storage interface {
	Save(filename string, reader io.Reader) (path string, err error)
	SaveAs(path string, reader io.Reader) error
	Delete(path string) error
	URL(path string) string
}

// LocalStorage stores files on the local filesystem.
type LocalStorage struct {
	BasePath string
	BaseURL  string
}

func NewLocalStorage(basePath, baseURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &LocalStorage{BasePath: basePath, BaseURL: baseURL}, nil
}

func (s *LocalStorage) Save(filename string, reader io.Reader) (string, error) {
	ext := filepath.Ext(filename)
	newName := uuid.New().String() + ext
	fullPath := filepath.Join(s.BasePath, newName)

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return newName, nil
}

func (s *LocalStorage) SaveAs(path string, reader io.Reader) error {
	normalizedPath := strings.TrimPrefix(strings.TrimSpace(path), "/")
	if normalizedPath == "" {
		return fmt.Errorf("path is required")
	}
	fullPath := filepath.Join(s.BasePath, normalizedPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (s *LocalStorage) Delete(filePath string) error {
	resolvedPath := strings.TrimSpace(filePath)
	if resolvedPath == "" {
		return nil
	}

	if parsedURL, err := url.Parse(resolvedPath); err == nil && parsedURL.Path != "" {
		resolvedPath = parsedURL.Path
	}

	resolvedPath = strings.TrimPrefix(resolvedPath, s.BaseURL)
	resolvedPath = strings.TrimPrefix(resolvedPath, "/")
	resolvedPath = path.Clean("/" + resolvedPath)
	resolvedPath = strings.TrimPrefix(resolvedPath, "/")
	if strings.Contains(resolvedPath, "..") {
		return fmt.Errorf("invalid path")
	}
	if resolvedPath == "." || resolvedPath == "" {
		return nil
	}

	fullPath := filepath.Join(s.BasePath, resolvedPath)
	if _, err := os.Stat(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	return os.Remove(fullPath)
}

func (s *LocalStorage) URL(path string) string {
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	rel := strings.TrimLeft(strings.TrimSpace(path), "/")
	if base == "" {
		return "/" + rel
	}
	return base + "/" + rel
}
