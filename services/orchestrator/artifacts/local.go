package artifacts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LocalFileStore persists one metadata JSON file per artifact under a durable
// volume. It replaces the single temp-file JSONStore for production startup;
// reopening the same directory restores all records.
type LocalFileStore struct {
	root string
	mu   sync.RWMutex
}

func NewLocalFileStore(root string) (*LocalFileStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, ErrStorePath
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create artifact store dir: %w", err)
	}
	return &LocalFileStore{root: root}, nil
}

func (s *LocalFileStore) Create(ctx context.Context, artifact ArtifactRecord) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	artifact.ID = strings.TrimSpace(artifact.ID)
	if artifact.ID == "" {
		return ErrInvalidArtifact
	}
	if err := safeArtifactID(artifact.ID); err != nil {
		return err
	}
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = nowUTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeLocked(artifact)
}

func (s *LocalFileStore) Get(ctx context.Context, id string) (*ArtifactRecord, bool, error) {
	if err := contextErr(ctx); err != nil {
		return nil, false, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, false, nil
	}
	if err := safeArtifactID(id); err != nil {
		return nil, false, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, err := s.readLocked(id)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &record, true, nil
}

func (s *LocalFileStore) ListByRun(ctx context.Context, runID string) ([]ArtifactRecord, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	runID = strings.TrimSpace(runID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	out := make([]ArtifactRecord, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		record, err := s.readLocked(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			continue
		}
		if runID == "" || record.RunID == runID {
			out = append(out, record)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (s *LocalFileStore) Delete(ctx context.Context, id string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if err := safeArtifactID(id); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.pathLocked(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *LocalFileStore) writeLocked(record ArtifactRecord) error {
	data, err := json.Marshal(cloneRecord(record))
	if err != nil {
		return err
	}
	path := s.pathLocked(record.ID)
	temp, err := os.CreateTemp(s.root, ".artifact-*")
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

func (s *LocalFileStore) readLocked(id string) (ArtifactRecord, error) {
	data, err := os.ReadFile(s.pathLocked(id))
	if err != nil {
		return ArtifactRecord{}, err
	}
	var record ArtifactRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return ArtifactRecord{}, err
	}
	return cloneRecord(record), nil
}

func (s *LocalFileStore) pathLocked(id string) string {
	return filepath.Join(s.root, id+".json")
}

func safeArtifactID(id string) error {
	if id == "." || id == ".." || strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") {
		return ErrInvalidArtifact
	}
	return nil
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func nowUTC() (t time.Time) { return time.Now().UTC() }
