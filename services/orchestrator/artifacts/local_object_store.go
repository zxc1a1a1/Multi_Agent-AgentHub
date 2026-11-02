package artifacts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LocalObjectStore is the single-node fallback blob store. It uses the same
// opaque-key boundary as S3ObjectStore so callers do not depend on the backend.
type LocalObjectStore struct {
	root string
	mu   sync.RWMutex
}

func NewLocalObjectStore(root string) (*LocalObjectStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, ErrStorePath
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create object store dir: %w", err)
	}
	return &LocalObjectStore{root: abs}, nil
}

func (s *LocalObjectStore) EnsureBucket(context.Context) error { return nil }

func (s *LocalObjectStore) Put(ctx context.Context, key string, data []byte, _ string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := validateObjectKey(key); err != nil {
		return err
	}
	if len(data) > maxObjectBytes {
		return fmt.Errorf("object exceeds size limit")
	}
	path, err := s.path(key)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".blob-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}

func (s *LocalObjectStore) Get(ctx context.Context, key string) ([]byte, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := validateObjectKey(key); err != nil {
		return nil, err
	}
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrObjectNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(data) > maxObjectBytes {
		return nil, fmt.Errorf("object exceeds size limit")
	}
	return data, nil
}

func (s *LocalObjectStore) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "", fmt.Errorf("presign is not supported by the local object store")
}

func (s *LocalObjectStore) Delete(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := validateObjectKey(key); err != nil {
		return err
	}
	path, err := s.path(key)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *LocalObjectStore) path(key string) (string, error) {
	clean := filepath.FromSlash(strings.TrimSpace(key))
	path := filepath.Join(s.root, clean)
	rel, err := filepath.Rel(s.root, path)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("invalid object key")
	}
	return path, nil
}

var _ ObjectStore = (*LocalObjectStore)(nil)
