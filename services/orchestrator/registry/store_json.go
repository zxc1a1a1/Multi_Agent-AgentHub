package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// storeFile is the top-level JSON structure persisted to disk.
type storeFile struct {
	Version int               `json:"version"`
	Agents  []RegisteredAgent `json:"agents"`
}

// JSONStore implements AgentStore backed by an atomic JSON file.
// It is safe for concurrent use via sync.RWMutex.
type JSONStore struct {
	mu   sync.RWMutex
	path string
}

// NewJSONStore creates a JSONStore at the given file path.
// The parent directory is created if it does not exist.
func NewJSONStore(path string) (*JSONStore, error) {
	if path == "" {
		return nil, errors.New("store path must not be empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create store directory %s: %w", dir, err)
	}
	return &JSONStore{path: path}, nil
}

// Load reads and returns all dynamic agent records.
// Returns an empty slice when the backing file does not exist.
func (s *JSONStore) Load(ctx context.Context) ([]RegisteredAgent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	all, err := s.loadLocked(ctx)
	if err != nil {
		// Missing file → empty slice.
		if errors.Is(err, os.ErrNotExist) {
			return []RegisteredAgent{}, nil
		}
		return nil, err
	}

	// Only return dynamic agents.
	var result []RegisteredAgent
	for _, a := range all {
		if a.Source == AgentSourceDynamic {
			result = append(result, a)
		}
	}
	if result == nil {
		result = []RegisteredAgent{}
	}
	return result, nil
}

// Upsert inserts or replaces a dynamic agent record by Name.
// Rejects agents with empty Name or Source == AgentSourceStatic.
func (s *JSONStore) Upsert(ctx context.Context, agent RegisteredAgent) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if agent.Name == "" {
		return errors.New("agent name must not be empty")
	}
	if agent.Source == AgentSourceStatic {
		return errors.New("static agents must not be persisted via AgentStore")
	}
	if agent.Source == "" {
		agent.Source = AgentSourceDynamic
	}

	now := time.Now()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	agent.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.loadLocked(ctx)
	if err != nil {
		// Missing file: start fresh. Corrupt or permission errors propagate.
		if errors.Is(err, os.ErrNotExist) {
			all = nil
		} else {
			return err
		}
	}

	// Upsert: replace existing or append.
	found := false
	for i, a := range all {
		if a.Name == agent.Name {
			// Preserve CreatedAt from the existing record.
			if !a.CreatedAt.IsZero() {
				agent.CreatedAt = a.CreatedAt
			}
			all[i] = agent
			found = true
			break
		}
	}
	if !found {
		all = append(all, agent)
	}

	return s.writeLocked(ctx, all)
}

// Delete removes a dynamic agent record by Name.
// Returns nil when the agent does not exist (no-op).
func (s *JSONStore) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if name == "" {
		return errors.New("agent name must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.loadLocked(ctx)
	if err != nil {
		// Missing file = nothing to delete.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	// Filter out the named agent.
	filtered := make([]RegisteredAgent, 0, len(all))
	for _, a := range all {
		if a.Name != name {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) == len(all) {
		// No match found — no-op.
		return nil
	}

	return s.writeLocked(ctx, filtered)
}

// loadLocked reads and decodes the JSON file. Does NOT acquire any lock —
// the caller must hold at least RLock.
func (s *JSONStore) loadLocked(ctx context.Context) ([]RegisteredAgent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("store file %s: %w", s.path, os.ErrNotExist)
		}
		return nil, fmt.Errorf("open store file %s: %w", s.path, err)
	}
	defer f.Close()

	var sf storeFile
	if err := json.NewDecoder(f).Decode(&sf); err != nil {
		return nil, fmt.Errorf("decode store file %s: %w", s.path, err)
	}

	if sf.Agents == nil {
		sf.Agents = []RegisteredAgent{}
	}
	return sf.Agents, nil
}

// writeLocked serializes and atomically writes only dynamic agents to disk
// via a uniquely-named temp file + rename. Does NOT acquire any lock —
// the caller must hold Lock.
func (s *JSONStore) writeLocked(ctx context.Context, agents []RegisteredAgent) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Filter: only persist dynamic agents. This prevents static entries that
	// were manually inserted into the JSON file from being written back.
	var filtered []RegisteredAgent
	for _, a := range agents {
		if a.Source == AgentSourceDynamic {
			filtered = append(filtered, a)
		}
	}
	if filtered == nil {
		filtered = []RegisteredAgent{}
	}

	dir := filepath.Dir(s.path)

	tmp, err := os.CreateTemp(dir, ".agents-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	cleanup := func() { os.Remove(tmpName) }

	sf := storeFile{Version: 1, Agents: filtered}
	if err := json.NewEncoder(tmp).Encode(&sf); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("encode store data: %w", err)
	}

	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file %s: %w", tmpName, err)
	}

	// Check context before committing the atomic rename.
	if err := ctx.Err(); err != nil {
		cleanup()
		return err
	}

	if err := os.Rename(tmpName, s.path); err != nil {
		cleanup()
		return fmt.Errorf("rename %s to %s: %w", tmpName, s.path, err)
	}

	return nil
}
