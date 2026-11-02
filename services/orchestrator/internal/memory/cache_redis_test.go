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
		t.Fatalf("forget: %v", err)
	}
	third, err := controller.Recall(ctx, scope, "cache", 5)
	if err != nil || len(third) != 0 {
		t.Fatalf("deleted memory must not be resurrected from cache: %#v %v", third, err)
	}
}

// Redis is a bounded cache, never the source of truth: keys must carry a TTL and
// an unreachable Redis has to degrade back to the SQL repository.
func TestIntegrationRedisCacheTTLAndOutageDegradesToSource(t *testing.T) {
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
	client := redis.NewClient(&redis.Options{Addr: redisAddr, DB: 3})
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
	principal := "cache-ttl-" + suffix
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
		Kind: "fact", LogicalKey: "ttl-key-" + suffix, Content: "ttl v1", SourceRefs: []string{"msg-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.Confirm(ctx, scope, entry.MemoryID, 1); err != nil {
		t.Fatal(err)
	}
	recalled, err := controller.Recall(ctx, scope, "ttl", 5)
	if err != nil || len(recalled) != 1 {
		t.Fatalf("recall: %#v err=%v", recalled, err)
	}
	keys, err := client.Keys(ctx, "memory:recall:*").Result()
	if err != nil || len(keys) == 0 {
		t.Fatalf("expected a recall cache key, got %v err=%v", keys, err)
	}
	for _, key := range keys {
		ttl, err := client.TTL(ctx, key).Result()
		if err != nil {
			t.Fatalf("ttl %s: %v", key, err)
		}
		if ttl <= 0 || ttl > 5*time.Minute {
			t.Fatalf("cache key %s ttl %v outside the documented five minute bound", key, ttl)
		}
	}

	// Same repository, Redis effectively unreachable: recall must still serve
	// authorized entries from SQL instead of failing the run.
	broken := redis.NewClient(&redis.Options{
		Addr: redisAddr, DB: 3, DialTimeout: time.Second,
		ReadTimeout: time.Microsecond, MaxRetries: 0,
	})
	defer broken.Close()
	brokenCache, err := NewRedisCache(broken)
	if err != nil {
		t.Fatal(err)
	}
	degraded, err := NewCached(memRepo, policy, brokenCache)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := degraded.Recall(ctx, scope, "ttl", 5)
	if err != nil {
		t.Fatalf("recall with unreachable redis must degrade: %v", err)
	}
	if len(entries) != 1 || entries[0].MemoryID != entry.MemoryID {
		t.Fatalf("degraded recall returned %#v", entries)
	}
}
