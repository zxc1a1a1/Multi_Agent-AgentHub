package httpapi

import (
	"context"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

type memoryAdminStore interface {
	ListEntries(ctx context.Context, scopeID, status string, limit int) ([]rtmemory.Entry, error)
	GetMutation(ctx context.Context, mutationID string) (*ledger.Mutation, error)
}
