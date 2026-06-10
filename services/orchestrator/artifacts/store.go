package artifacts

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ArtifactRecord is metadata only. Large binary/file contents are deliberately
// not stored here; Phase 8 records metadata/card information only.
type ArtifactRecord struct {
	ID           string         `json:"id"`
	RunID        string         `json:"runId,omitempty"`
	TaskID       string         `json:"taskId,omitempty"`
	MessageID    string         `json:"messageId,omitempty"`
	Name         string         `json:"name,omitempty"`
	Kind         string         `json:"kind,omitempty"`
	MimeType     string         `json:"mimeType,omitempty"`
	Size         int64          `json:"size,omitempty"`
	StoragePath  string         `json:"storagePath,omitempty"`
	DownloadPath string         `json:"downloadPath,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type Store interface {
	Create(ctx context.Context, artifact ArtifactRecord) error
	Get(ctx context.Context, id string) (*ArtifactRecord, bool, error)
	ListByRun(ctx context.Context, runID string) ([]ArtifactRecord, error)
	Delete(ctx context.Context, id string) error
}

var (
	ErrInvalidArtifact = errors.New("invalid artifact")
	ErrStorePath       = errors.New("artifact store path is required")
)

type JSONStore struct {
	path string
	mu   sync.RWMutex
	data map[string]ArtifactRecord
}

func NewJSONStore(path string) (*JSONStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, ErrStorePath
	}
	s := &JSONStore{path: path, data: map[string]ArtifactRecord{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *JSONStore) Create(ctx context.Context, artifact ArtifactRecord) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return ErrStorePath
	}
	artifact.ID = strings.TrimSpace(artifact.ID)
	if artifact.ID == "" {
		return ErrInvalidArtifact
	}
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = time.Now().UTC()
	}
	artifact.Metadata = cloneMap(artifact.Metadata)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[artifact.ID] = artifact
	return s.saveLocked()
}

func (s *JSONStore) Get(ctx context.Context, id string) (*ArtifactRecord, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if s == nil {
		return nil, false, ErrStorePath
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, false, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.data[id]
	if !ok {
		return nil, false, nil
	}
	cloned := cloneRecord(rec)
	return &cloned, true, nil
}

func (s *JSONStore) ListByRun(ctx context.Context, runID string) ([]ArtifactRecord, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil {
		return nil, ErrStorePath
	}
	runID = strings.TrimSpace(runID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ArtifactRecord, 0)
	for _, rec := range s.data {
		if runID == "" || rec.RunID == runID {
			out = append(out, cloneRecord(rec))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *JSONStore) Delete(ctx context.Context, id string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return ErrStorePath
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return s.saveLocked()
}

func (s *JSONStore) load() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil
	}
	var records []ArtifactRecord
	if err := json.Unmarshal(b, &records); err != nil {
		return err
	}
	for _, rec := range records {
		if strings.TrimSpace(rec.ID) == "" {
			continue
		}
		s.data[rec.ID] = cloneRecord(rec)
	}
	return nil
}

func (s *JSONStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	records := make([]ArtifactRecord, 0, len(s.data))
	for _, rec := range s.data {
		records = append(records, cloneRecord(rec))
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	b, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

func cloneRecord(in ArtifactRecord) ArtifactRecord {
	in.Metadata = cloneMap(in.Metadata)
	return in
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
