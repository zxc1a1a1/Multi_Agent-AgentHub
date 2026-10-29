package httpapi

import (
	"context"
	"log"
	"os"
	"strings"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/planner"
)

// recallMemory injects authorized long-term memory before the model call.
// Recall failures degrade to no memory; they never apply stale cache data.
func (s *Server) recallMemory(ctx context.Context, userID, query string) []planner.MemorySnippet {
	if s == nil {
		return nil
	}
	s.memoryMu.RLock()
	controller := s.memoryController
	s.memoryMu.RUnlock()
	if controller == nil || strings.TrimSpace(query) == "" {
		return nil
	}
	env := strings.TrimSpace(os.Getenv("AGENTHUB_ENV"))
	if env == "" {
		env = "dev"
	}
	principal := strings.TrimSpace(userID)
	if principal == "" {
		principal = "demo-user"
	}
	scope := rtmemory.Scope{Env: env, WorkspaceID: "default", PrincipalID: principal}
	entries, err := controller.Recall(ctx, scope, query, 5)
	if err != nil {
		log.Printf("orchestrator: memory recall skipped: %v", err)
		return nil
	}
	snippets := make([]planner.MemorySnippet, 0, len(entries))
	for _, entry := range entries {
		snippets = append(snippets, planner.MemorySnippet{
			MemoryID: entry.MemoryID, Revision: entry.Revision,
			Content: entry.Content, SourceRefs: entry.SourceRefs,
		})
	}
	return snippets
}
