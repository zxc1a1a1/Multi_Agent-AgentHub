package pruning

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/adk"
)

// R1-02 短期上下文边界测试（pruning 侧）。
//
// 覆盖的真实生产实现：
//   - PreparePinnedMessages（context_prepare.go:16）head/tail 窗口 + pinned 覆盖
//   - Orchestrator 实际使用的窗口 head=20 / tail=4（handler_run_stream.go:1133）
//   - KeepPinnedWindowPruner 的 HardLimit 稳定错误（pruner.go:411）
//   - PruningPlugin.BeforeGenerate 把 pruner 错误向上抛出，模型调用不会发生
//     （plugin.go:44）
//   - EnsurePairedFunctionCalls 补齐缺失的 tool result（pruner.go:137）
//
// 证据只记录条数/字符数/丢弃数，不记录正文。

func r102pRecord(t *testing.T, target string, counts map[string]any, errClass string) {
	t.Helper()
	path := os.Getenv("AGENTHUB_EVIDENCE_JSONL")
	if path == "" {
		return
	}
	status := "ok"
	if errClass != "" {
		status = "error"
	}
	row := map[string]any{
		"schemaVersion": 2,
		"evidenceId":    fmt.Sprintf("ev_r1_02_%s", t.Name()),
		"testId":        "R1-02",
		"planTaskId":    "R1-02",
		"layer":         "orchestrator",
		"actor":         "orchestrator",
		// context_trim 是 §5.2 action 枚举的显式扩展，已在 summary.md 说明。
		"action":     "context_trim",
		"target":     target,
		"status":     status,
		"errorCode":  errClass,
		"counts":     counts,
		"detail":     map[string]any{"errorClass": errClass},
		"recordedAt": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Logf("evidence append failed: %v", err)
		return
	}
	defer file.Close()
	encoded, _ := json.Marshal(row)
	_, _ = file.Write(append(encoded, '\n'))
}

func r102pChars(msgs []MessageWithID) int {
	total := 0
	for _, m := range msgs {
		total += len(m.Text)
	}
	return total
}

// TestR102PreparePinnedMessagesWindowIsExact 用 Orchestrator 真实使用的
// head=20/tail=4 断言窗口边界：30 条消息必须保留 index 0..19 与 26..29。
func TestR102PreparePinnedMessagesWindowIsExact(t *testing.T) {
	const head, tail = 20, 4 // handler_run_stream.go:1133

	messages := make([]MessageWithID, 30)
	for i := range messages {
		messages[i] = MessageWithID{ID: fmt.Sprintf("m-%02d", i), Role: "user", Text: fmt.Sprintf("body-%02d", i)}
	}

	filtered, missing := PreparePinnedMessages(messages, nil, head, tail)
	if len(missing) != 0 {
		t.Fatalf("no pins were requested, got missing=%v", missing)
	}

	want := map[string]bool{}
	for i := 0; i < head; i++ {
		want[fmt.Sprintf("m-%02d", i)] = true
	}
	for i := len(messages) - tail; i < len(messages); i++ {
		want[fmt.Sprintf("m-%02d", i)] = true
	}
	if len(filtered) != len(want) {
		t.Fatalf("window kept %d messages, want %d", len(filtered), len(want))
	}
	for _, m := range filtered {
		if !want[m.ID] {
			t.Fatalf("unexpected message %q inside the window", m.ID)
		}
	}
	// 中段 6 条（index 20..25）必须被裁掉。
	if dropped := len(messages) - len(filtered); dropped != 6 {
		t.Fatalf("dropped %d, want 6", dropped)
	}
	r102pRecord(t, "pruning.PreparePinnedMessages", map[string]any{
		"itemsIn": len(messages), "itemsOut": len(filtered),
		"itemsDropped": len(messages) - len(filtered),
		"charsIn":      r102pChars(messages), "charsOut": r102pChars(filtered),
		"keepFirst": head, "keepLast": tail,
	}, "")
}

// TestR102PreparePinnedMessagesIsDeterministicAndOrderStable 断言窗口选择可重复，
// 并且输出保持原始相对顺序（前端历史重放依赖这一点）。
func TestR102PreparePinnedMessagesIsDeterministicAndOrderStable(t *testing.T) {
	messages := make([]MessageWithID, 40)
	for i := range messages {
		messages[i] = MessageWithID{ID: fmt.Sprintf("m-%02d", i), Role: "user", Text: fmt.Sprintf("body-%02d", i)}
	}
	pins := []string{"m-30", "m-05"}

	const head, tail = 10, 5
	var previous []string
	for run := 0; run < 5; run++ {
		filtered, _ := PreparePinnedMessages(messages, pins, head, tail)
		ids := make([]string, 0, len(filtered))
		for _, m := range filtered {
			ids = append(ids, m.ID)
		}
		if previous != nil && strings.Join(previous, ",") != strings.Join(ids, ",") {
			t.Fatalf("window selection is not deterministic: run %d differs", run)
		}
		previous = ids
	}
	// head 覆盖 m-00..m-09（含 pin m-05），tail 覆盖 m-35..m-39，
	// 中段 pin m-30 额外保留一条。
	if len(previous) != head+tail+1 {
		t.Fatalf("expected %d kept messages, got %d: %v", head+tail+1, len(previous), previous)
	}
	for i := 1; i < len(previous); i++ {
		if previous[i-1] >= previous[i] {
			t.Fatalf("original order not preserved: %v", previous)
		}
	}
}

// TestR102PreparePinnedMessagesKeepsPinnedOutsideWindowWhileTrimming 断言裁剪
// 只会移除未 pinned 的条目，pinned 约束不会被静默丢弃。
func TestR102PreparePinnedMessagesKeepsPinnedOutsideWindowWhileTrimming(t *testing.T) {
	messages := make([]MessageWithID, 50)
	for i := range messages {
		messages[i] = MessageWithID{ID: fmt.Sprintf("m-%02d", i), Role: "user", Text: fmt.Sprintf("body-%02d", i)}
	}
	// 两条落在中段窗口之外（index 10..44），必须被保留。
	messages[22] = MessageWithID{ID: "pin-a", Role: "user", Text: strings.Repeat("a", 500)}
	messages[33] = MessageWithID{ID: "pin-b", Role: "assistant", Text: strings.Repeat("b", 500)}

	filtered, missing := PreparePinnedMessages(messages, []string{"pin-a", "pin-b", "pin-ghost"}, 5, 3)
	if len(missing) != 1 || missing[0] != "pin-ghost" {
		t.Fatalf("unknown pin must be reported as missing, got %v", missing)
	}

	seen := map[string]bool{}
	for _, m := range filtered {
		seen[m.ID] = true
	}
	for _, id := range []string{"pin-a", "pin-b"} {
		if !seen[id] {
			t.Fatalf("pinned %q was dropped while trimming", id)
		}
	}
	for _, m := range messages {
		if m.ID == "m-20" && seen["m-20"] {
			t.Fatal("unpinned middle message should have been trimmed")
		}
	}
	dropped := len(messages) - len(filtered)
	if dropped <= 0 {
		t.Fatal("no trimming happened; the corpus does not exercise the window")
	}
	for _, m := range filtered {
		if !strings.HasPrefix(m.ID, "m-") {
			continue
		}
		idx := 0
		if _, err := fmt.Sscanf(m.ID, "m-%d", &idx); err != nil {
			t.Fatalf("bad id %q", m.ID)
		}
		if idx >= 5 && idx < 47 {
			t.Fatalf("unpinned middle message %q survived the trim", m.ID)
		}
	}
	r102pRecord(t, "pruning.PreparePinnedMessages", map[string]any{
		"itemsIn": len(messages), "itemsOut": len(filtered), "itemsDropped": dropped,
		"charsIn": r102pChars(messages), "charsOut": r102pChars(filtered),
		"pinnedPreserved": 2, "missingPins": len(missing),
	}, "")
}

// TestR102HardLimitRejectsPinnedOverflowBeforeModelCall 是 M-04 的核心断言：
// pinned + 窗口选择超过 HardLimit 时必须返回稳定错误，并且
// PruningPlugin 把错误透传给 BeforeGenerate，模型生成不会发生。
func TestR102HardLimitRejectsPinnedOverflowBeforeModelCall(t *testing.T) {
	contents := make([]*adk.Content, 0, 6)
	for i := 0; i < 6; i++ {
		contents = append(contents, &adk.Content{
			Role:  adk.RoleUser,
			Parts: []adk.Part{adk.TextPart{Text: fmt.Sprintf("pinned-%d", i)}},
		})
	}
	req := &adk.GenerateRequest{Contents: contents}
	pinned := map[int]bool{0: true, 1: true, 2: true, 3: true, 4: true, 5: true}

	// 恰好在 hard limit 之内：允许通过。
	inside, err := NewKeepPinnedWindowPruner(0, 0, pinned, len(contents)).Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("selection of %d items at limit %d must pass, got %v", len(contents), len(contents), err)
	}
	if len(inside.Contents) != len(contents) {
		t.Fatalf("all pinned items must be kept, got %d", len(inside.Contents))
	}

	// 超出 1 条：必须明确失败，且不返回被裁剪后的“看起来可用”的上下文。
	pruner := NewKeepPinnedWindowPruner(0, 0, pinned, len(contents)-1)
	out, err := pruner.Prune(context.Background(), req)
	if err == nil {
		t.Fatal("pinned context over the hard limit must fail")
	}
	if err.Error() != "pinned context exceeds hard limit" {
		t.Fatalf("error text is not stable: %q", err.Error())
	}
	if out != nil {
		t.Fatal("overflow must not return a pruned request")
	}

	plugin := NewPruningPlugin(pruner)
	if err := plugin.BeforeGenerate(context.Background(), nil, req); err == nil {
		t.Fatal("BeforeGenerate must surface the hard-limit error so generation never starts")
	} else if !strings.Contains(err.Error(), "pinned context exceeds hard limit") {
		t.Fatalf("unexpected error from BeforeGenerate: %v", err)
	}

	// 同一个请求重复判定，错误必须完全一致。
	for i := 0; i < 3; i++ {
		_, again := pruner.Prune(context.Background(), req)
		if again == nil || again.Error() != err.Error() {
			t.Fatalf("hard-limit error is not deterministic: %v", again)
		}
	}
	r102pRecord(t, "pruning.KeepPinnedWindowPruner", map[string]any{
		"itemsIn": len(contents), "itemsOut": 0, "itemsDropped": len(contents),
		"pinnedCount": len(pinned), "hardLimit": len(contents) - 1,
	}, "pinned_context_exceeds_hard_limit")
}

// TestR102HardLimitOnlyCountsUnpinnedDroppableContent 断言 HardLimit 的语义是
// “保留条目数”，而不是“输入条目数”：窗口外的条目被丢弃后不应再触发错误。
func TestR102HardLimitOnlyCountsKeptSelection(t *testing.T) {
	contents := make([]*adk.Content, 0, 10)
	for i := 0; i < 10; i++ {
		contents = append(contents, &adk.Content{
			Role:  adk.RoleUser,
			Parts: []adk.Part{adk.TextPart{Text: fmt.Sprintf("body-%d", i)}},
		})
	}
	req := &adk.GenerateRequest{Contents: contents}

	// head=2, tail=2 => 4 条被保留；limit=4 应该通过。
	out, err := NewKeepPinnedWindowPruner(2, 2, nil, 4).Prune(context.Background(), req)
	if err != nil {
		t.Fatalf("kept selection equals the limit, must pass: %v", err)
	}
	if len(out.Contents) != 4 {
		t.Fatalf("expected 4 kept contents, got %d", len(out.Contents))
	}
	// limit=3 而保留 4 条，必须明确失败。
	if _, err := NewKeepPinnedWindowPruner(2, 2, nil, 3).Prune(context.Background(), req); err == nil {
		t.Fatal("kept selection over the limit must fail")
	}
	r102pRecord(t, "pruning.KeepPinnedWindowPruner", map[string]any{
		"itemsIn": 10, "itemsOut": 4, "itemsDropped": 6, "hardLimit": 4,
	}, "")
}

// TestR102KeepEndsWindowPrunerHeadTailSelection 断言 launcher 默认窗口
// （head=4, tail=12, launcher.go:17-18）的确定性选择。
func TestR102KeepEndsWindowPrunerHeadTailSelection(t *testing.T) {
	const head, tail = 4, 12 // launcher.go:17-18

	contents := make([]*adk.Content, 0, 20)
	for i := 0; i < 20; i++ {
		contents = append(contents, &adk.Content{
			Role:  adk.RoleUser,
			Parts: []adk.Part{adk.TextPart{Text: fmt.Sprintf("c-%02d", i)}},
		})
	}
	// 中段放一条 system，必须被强制保留。
	contents[15] = &adk.Content{Role: adk.RoleSystem, Parts: []adk.Part{adk.TextPart{Text: "system-must-survive"}}}

	out, err := NewKeepEndsWindowPruner(head, tail).Prune(context.Background(), &adk.GenerateRequest{Contents: contents})
	if err != nil {
		t.Fatal(err)
	}
	// index 0..3 (head) + index 8..19 (tail) + index 15 (system, already in tail)。
	if len(out.Contents) != 16 {
		t.Fatalf("expected 16 kept contents, got %d", len(out.Contents))
	}
	var texts []string
	for _, c := range out.Contents {
		tp, ok := c.Parts[0].(adk.TextPart)
		if !ok {
			t.Fatalf("unexpected part type in %T", c)
		}
		texts = append(texts, tp.Text)
	}
	joined := strings.Join(texts, ",")
	for _, want := range []string{"c-00", "c-03", "c-08", "c-19", "system-must-survive"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("%q must be inside the head/tail window, got %s", want, joined)
		}
	}
	for _, gone := range []string{"c-04", "c-07"} {
		if strings.Contains(joined, gone) {
			t.Fatalf("%q is outside the window and must be pruned", gone)
		}
	}
	r102pRecord(t, "pruning.KeepEndsWindowPruner", map[string]any{
		"itemsIn": 20, "itemsOut": len(out.Contents),
		"itemsDropped": 20 - len(out.Contents), "keepFirst": head, "keepLast": tail,
	}, "")
}

// TestR102WindowPruneThenPairingKeepsEveryCallPaired 证明窗口裁剪之后再做
// 配对补齐，仍然不会出现“半对”工具调用。
func TestR102WindowPruneThenPairingKeepsEveryCallPaired(t *testing.T) {
	contents := []*adk.Content{
		{Role: adk.RoleSystem, Parts: []adk.Part{adk.TextPart{Text: "sys"}}},
	}
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("call-%d", i)
		contents = append(contents,
			&adk.Content{Role: adk.RoleAssistant, Parts: []adk.Part{
				adk.ToolCallPart{ID: id, Name: "workspace.read", Arguments: json.RawMessage(`{}`)},
			}},
			&adk.Content{Role: adk.RoleTool, Parts: []adk.Part{
				adk.ToolResultPart{CallID: id, Name: "workspace.read", Content: "ok"},
			}},
		)
	}
	req := &adk.GenerateRequest{Contents: contents}

	// 窗口只保留 head=1(即 index 0)、tail=2，中段工具对必须成对消失。
	out, err := NewKeepEndsWindowPruner(1, 2).Prune(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	paired := EnsurePairedFunctionCalls(out.Contents)

	calls := map[string]bool{}
	results := map[string]bool{}
	for _, c := range paired {
		if c == nil {
			t.Fatal("nil content after pairing")
		}
		for _, part := range c.Parts {
			switch p := part.(type) {
			case adk.ToolCallPart:
				calls[p.ID] = true
			case adk.ToolResultPart:
				results[p.CallID] = true
			}
		}
	}
	for id := range calls {
		if !results[id] {
			t.Fatalf("call %s has no result after EnsurePairedFunctionCalls", id)
		}
	}
	for id := range results {
		if !calls[id] {
			t.Fatalf("result %s is orphaned after window pruning", id)
		}
	}
	r102pRecord(t, "pruning.EnsurePairedFunctionCalls", map[string]any{
		"itemsIn": len(contents), "itemsOut": len(paired),
		"toolCalls": len(calls), "toolResults": len(results),
	}, "")
}

// TestR102OrphanResultAfterTrimIsNotSilentlyRewrittenAsValid 断言：窗口裁剪留下的
// 孤立 tool result 不会被当成有效证据继续存在（EnsurePairedFunctionCalls 只补
// 缺失的 result，不会为孤立 result 伪造 call）。
func TestR102OrphanResultIsNeverBackedByAFabricatedCall(t *testing.T) {
	contents := []*adk.Content{
		{Role: adk.RoleUser, Parts: []adk.Part{adk.TextPart{Text: "hello"}}},
		{Role: adk.RoleTool, Parts: []adk.Part{
			adk.ToolResultPart{CallID: "ghost-call", Name: "apply_patch", Content: "orphan"},
		}},
	}
	out := EnsurePairedFunctionCalls(contents)

	var callCount, resultCount int
	for _, c := range out {
		for _, part := range c.Parts {
			switch part.(type) {
			case adk.ToolCallPart:
				callCount++
			case adk.ToolResultPart:
				resultCount++
			}
		}
	}
	if callCount != 0 {
		t.Fatalf("EnsurePairedFunctionCalls fabricated %d tool call(s) for an orphan result", callCount)
	}
	if resultCount != 1 {
		t.Fatalf("the orphan result must be left untouched here (trimming it is BuildContext's job), got %d", resultCount)
	}

	// 反过来：assistant 发起但缺 result 时，必须补一条明确的错误 result，
	// 这样 harness.BuildContext 才把它视为有主的 result 而不是凭空调用。
	withCall := []*adk.Content{{
		Role:  adk.RoleAssistant,
		Parts: []adk.Part{adk.ToolCallPart{ID: "solo-call", Name: "workspace.read", Arguments: json.RawMessage(`{}`)}},
	}}
	paired := EnsurePairedFunctionCalls(withCall)
	var found bool
	for _, c := range paired {
		for _, part := range c.Parts {
			if r, ok := part.(adk.ToolResultPart); ok && r.CallID == "solo-call" && r.IsError {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("missing tool result must be synthesised as an error, not dropped")
	}
}
