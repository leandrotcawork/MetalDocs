package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Store struct {
	rootPath string
}

func NewStore(rootPath string) *Store {
	return &Store{rootPath: rootPath}
}

func (s *Store) Save(_ context.Context, storageKey string, content []byte) error {
	target := filepath.Join(s.rootPath, filepath.FromSlash(storageKey))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("mkdir attachment path: %w", err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		return fmt.Errorf("write attachment content: %w", err)
	}
	return nil
}

func (s *Store) Open(_ context.Context, storageKey string) (io.ReadCloser, error) {
	target := filepath.Join(s.rootPath, filepath.FromSlash(storageKey))
	file, err := os.Open(target)
	if err != nil {
		return nil, fmt.Errorf("open attachment content: %w", err)
	}
	return file, nil
}

func (s *Store) Delete(_ context.Context, storageKey string) error {
	target := filepath.Join(s.rootPath, filepath.FromSlash(storageKey))
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete attachment content: %w", err)
	}
	return nil
}
