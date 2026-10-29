package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

const maxArtifactBytes = 1024 * 1024

type Artifact struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	MimeType  string    `json:"mimeType"`
	Content   string    `json:"content"`
	Size      int       `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

type ArtifactStore interface {
	Save(ctx context.Context, artifact Artifact) (string, error)
}

type InMemoryArtifactStore struct {
	mu   sync.Mutex
	byID map[string]Artifact
}

func NewInMemoryArtifactStore() *InMemoryArtifactStore {
	return &InMemoryArtifactStore{byID: map[string]Artifact{}}
}

func (s *InMemoryArtifactStore) Save(_ context.Context, artifact Artifact) (string, error) {
	if artifact.ID == "" {
		var buf [8]byte
		if _, err := rand.Read(buf[:]); err != nil {
			return "", err
		}
		artifact.ID = "art_" + hex.EncodeToString(buf[:])
	}
	artifact.Size = len(artifact.Content)
	artifact.CreatedAt = time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[artifact.ID] = artifact
	return artifact.ID, nil
}

func (s *InMemoryArtifactStore) Get(id string) (Artifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.byID[id]
	return a, ok
}

func ArtifactPublishTool(store ArtifactStore) *workspaceTool {
	schema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"mimeType":{"type":"string"},"content":{"type":"string"}},"required":["name","content"],"additionalProperties":false}`)
	d := Descriptor{
		Name: "artifact.publish", Version: "1", Description: "Archive a bounded artifact and return its internal id.",
		InputSchema: schema, ReadOnly: false, Destructive: false, Approval: ApprovalConfirm,
		Timeout: 15 * time.Second, MaxOutputBytes: 4096, Provider: "native",
	}
	return &workspaceTool{name: d.Name, description: d.Description, schema: schema, descriptor: &d,
		run: func(ctx context.Context, raw json.RawMessage) (*adk.ToolResult, error) {
			if store == nil {
				return nil, fmt.Errorf("artifact store is not configured")
			}
			var args struct {
				Name     string `json:"name"`
				MimeType string `json:"mimeType"`
				Content  string `json:"content"`
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return nil, err
			}
			if args.Name == "" || len(args.Content) == 0 || len(args.Content) > maxArtifactBytes {
				return nil, fmt.Errorf("name and bounded content are required")
			}
			id, err := store.Save(ctx, Artifact{Name: args.Name, MimeType: args.MimeType, Content: args.Content})
			if err != nil {
				return nil, err
			}
			return jsonResult(map[string]any{"artifactId": id, "size": len(args.Content)})
		}}
}
