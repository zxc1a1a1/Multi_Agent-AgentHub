package memory

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	rtmemory "github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/memory"
	ledger "github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql"
)

func TestIntegrationRedisRecallCacheUsesEpoch(t *testing.T) {
	dsn := os.Getenv("AGENTHUB_MYSQL_TEST_DSN")
	redisAddr := os.Getenv("AGENTHUB_REDIS_TEST_ADDR")
	if dsn == "" || redisAddr == "" {
		t.Skip("SKIP real MySQL+Redis integration: set AGENTHUB_MYSQL_TEST_DSN and AGENTHUB_REDIS_TEST_ADDR")
	}
	ctx := context.Background()
	db, err := ledger.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := ledger.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	cache, err := NewRedisCache(client)
	if err != nil {
		t.Fatal(err)
	}
	memRepo := ledger.NewMemoryRepository(db)
	policy := rtmemory.DefaultPolicy()
	policy.Mode = rtmemory.ModeScopedLongTerm
	policy.Read, policy.Propose, policy.Confirm, policy.Forget = true, true, true, true
	controller, err := NewCached(memRepo, policy, cache)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	principal := "cache-" + suffix
	scope := rtmemory.Scope{Env: "test", WorkspaceID: "ws", PrincipalID: principal}
	t.Cleanup(func() {
		keys, _ := client.Keys(context.Background(), "memory:recall:*").Result()
		if len(keys) > 0 {
			_ = client.Del(context.Background(), keys...).Err()
		}
		_, _ = db.Exec("DELETE FROM memory_revisions WHERE memory_id LIKE ?", "mem_"+suffix+"%")
		_, _ = db.Exec("DELETE FROM memory_entries WHERE scope_id IN (SELECT scope_id FROM memory_scopes WHERE principal_id = ?)", principal)
		_, _ = db.Exec("DELETE FROM memory_mutations WHERE scope_id IN (SELECT scope_id FROM memory_scopes WHERE principal_id = ?)", principal)
		_, _ = db.Exec("DELETE FROM memory_scopes WHERE principal_id = ?", principal)
	})

	entry, err := controller.Propose(ctx, scope, rtmemory.Entry{
		Kind: "fact", LogicalKey: "cache-key-" + suffix, Content: "cache v1", SourceRefs: []string{"msg-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.Confirm(ctx, scope, entry.MemoryID, 1); err != nil {
		t.Fatal(err)
	}
	first, err := controller.Recall(ctx, scope, "cache", 5)
	if err != nil || len(first) != 1 || first[0].Content != "cache v1" {
		t.Fatalf("first recall: %#v %v", first, err)
	}
	keys, _ := client.Keys(ctx, "memory:recall:*").Result()
	if len(keys) == 0 {
		t.Fatal("recall result was not cached in Redis")
	}

	if _, err := controller.Update(ctx, scope, entry.MemoryID, "cache v2", 1); err != nil {
		t.Fatal(err)
	}
	second, err := controller.Recall(ctx, scope, "cache", 5)
	if err != nil || len(second) != 1 || second[0].Content != "cache v2" {
		t.Fatalf("epoch change must bypass old cache: %#v %v", second, err)
	}
	if err := controller.Forget(ctx, scope, entry.MemoryID); err != nil {
		t.Fatal(err)
	}
	third, err := controller.Recall(ctx, scope, "cache", 5)
	if err != nil || len(third) != 0 {
		t.Fatalf("deleted memory must not be resurrected from cache: %#v %v", third, err)
	}
}
