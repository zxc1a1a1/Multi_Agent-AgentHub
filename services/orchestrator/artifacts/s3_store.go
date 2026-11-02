package artifacts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// S3Store persists artifact metadata as one JSON object per artifact so
// multiple Orchestrator replicas can read the same records through S3/MinIO.
type S3Store struct {
	objects ObjectStore
	prefix  string
}

func NewS3Store(objects ObjectStore, prefix string) (*S3Store, error) {
	if objects == nil {
		return nil, fmt.Errorf("object store is required")
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix != "" {
		prefix += "/"
	}
	return &S3Store{objects: objects, prefix: prefix}, nil
}

func (s *S3Store) Create(ctx context.Context, artifact ArtifactRecord) error {
	artifact.ID = strings.TrimSpace(artifact.ID)
	if artifact.ID == "" {
		return ErrInvalidArtifact
	}
	if err := safeArtifactID(artifact.ID); err != nil {
		return err
	}
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = time.Now().UTC()
	}
	data, err := json.Marshal(artifact)
	if err != nil {
		return err
	}
	return s.objects.Put(ctx, s.key(artifact.ID), data, "application/json")
}

func (s *S3Store) Get(ctx context.Context, id string) (*ArtifactRecord, bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, false, nil
	}
	if err := safeArtifactID(id); err != nil {
		return nil, false, nil
	}
	data, err := s.objects.Get(ctx, s.key(id))
	if errors.Is(err, ErrObjectNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var record ArtifactRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, false, err
	}
	return &record, true, nil
}

func (s *S3Store) ListByRun(ctx context.Context, runID string) ([]ArtifactRecord, error) {
	runID = strings.TrimSpace(runID)
	keys, err := s.objects.List(ctx, s.prefix)
	if err != nil {
		return nil, err
	}
	out := make([]ArtifactRecord, 0)
	for _, key := range keys {
		if !strings.HasSuffix(key, ".json") {
			continue
		}
		data, err := s.objects.Get(ctx, key)
		if err != nil {
			continue
		}
		var record ArtifactRecord
		if err := json.Unmarshal(data, &record); err != nil {
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

func (s *S3Store) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if err := safeArtifactID(id); err != nil {
		return err
	}
	return s.objects.Delete(ctx, s.key(id))
}

func (s *S3Store) key(id string) string {
	return s.prefix + id + ".json"
}

var _ Store = (*S3Store)(nil)
