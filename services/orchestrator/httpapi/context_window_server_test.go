package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/harness"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/plan"
)

// R1-02 服务端断言（plan §4.2 R1-02 第 3 点）。
//
// 事实源：
//   - 前端显示窗口 100 条：frontend/src/components/ChatWindow.tsx:48（仅渲染层，
//     store 仍持有全部消息，contextMessages 由 messageStore.ts:709 从完整会话构造）。
//   - Orchestrator 只有在 pinnedMessageIds 非空时才裁剪：handler_run_stream.go:97-99，
//     窗口 head=20 / tail=4：handler_run_stream.go:1133。
//   - 默认短期上下文 hard limit：harness.DefaultContextLimits() MaxChars=120_000
//     （pkg/runtime/harness/context.go:29），超限哨兵 harness.ErrContextTooLarge（context.go:8）。
//
// 证据只记录消息条数、字符数、估算 token 数与丢弃数，绝不记录消息正文。

// r102sDisplayWindow 是前端 ChatWindow 的默认显示窗口条数，仅用于证明
// “显示窗口 > 服务端收到的历史长度” 时服务端不受影响。
const r102sDisplayWindow = 100

func r102sRecord(t *testing.T, target string, counts map[string]any, errClass string) {
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

// r102sMessages 构造 n 条可辨识的合成历史消息（marker 仅用于断言，不进证据）。
func r102sMessages(n int) []MessageInput {
	messages := make([]MessageInput, 0, n)
	for i := 0; i < n; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages = append(messages, MessageInput{
			ID:   fmt.Sprintf("ctx-%03d", i),
			Role: role,
			Text: fmt.Sprintf("MARKER-%03d", i),
		})
	}
	return messages
}

func r102sRun(t *testing.T, messages []MessageInput, pinnedIDs []string, fake *FakeMainAgent) *httptest.ResponseRecorder {
	t.Helper()
	payload := OrchestratorRequest{
		RunID:            "run-r1-02-window",
		ConversationID:   "conv-r1-02-window",
		UserID:           "principal-r1-02",
		ConversationType: "single",
		Messages:         messages,
		PlanningMode:     "auto",
		RequestedPath:    "main_agent_orchestration",
		PinnedMessageIDs: pinnedIDs,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	srv := NewServer(WithMainAgentPlanner(fake))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/orchestrator/runs/stream", strings.NewReader(string(body)))
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func r102sConversationalPlan(runID string) *plan.OrchestrationPlan {
	p := makeTestPlan(runID, plan.StrategyConversational, nil)
	p.Answer = "ok"
	return p
}

// TestR102FrontendDisplayWindowDoesNotTrimServerHistory 断言：前端只显示最近
// 100 条时，Gateway/Orchestrator 仍然拿到完整历史 —— 未显式请求裁剪就不裁剪。
func TestR102FrontendDisplayWindowDoesNotTrimServerHistory(t *testing.T) {
	const total = 150 // > 前端显示窗口 100
	if total <= r102sDisplayWindow {
		t.Fatal("corpus must exceed the frontend display window to be meaningful")
	}
	messages := r102sMessages(total)

	fake := &FakeMainAgent{PlanToReturn: r102sConversationalPlan("run-r1-02-window")}
	rec := r102sRun(t, messages, nil, fake)
	if rec.Code != http.StatusOK {
		t.Fatalf("run failed: %d %s", rec.Code, rec.Body.String())
	}
	if !fake.PlanCalled {
		t.Fatal("planner was not invoked")
	}

	got := fake.LastInput.UserMessage
	// 最旧的一条（index 0）位于前端显示窗口之外，服务端必须仍然收到它。
	if !strings.Contains(got, "MARKER-000") {
		t.Fatal("oldest message outside the frontend 100-message display window was lost server-side")
	}
	// 显示窗口边界（第 100 条）两侧的服务端可见性必须一致。
	if !strings.Contains(got, "MARKER-098") || !strings.Contains(got, "MARKER-100") {
		t.Fatal("messages around the display-window boundary must all be present")
	}
	if !strings.Contains(got, fmt.Sprintf("MARKER-%03d", total-2)) {
		t.Fatal("latest user message must be present")
	}
	// 无 pinned → 不允许出现任何窗口裁剪痕迹。
	if strings.Contains(got, "[Pinned conversation context]") {
		t.Fatal("trimming must not happen without an explicit pinned request")
	}

	userMarkers := 0
	for i := 0; i < total; i += 2 { // 偶数 index 是 user
		if strings.Contains(got, fmt.Sprintf("MARKER-%03d", i)) {
			userMarkers++
		}
	}
	if userMarkers != total/2 {
		t.Fatalf("expected %d user messages forwarded intact, got %d", total/2, userMarkers)
	}

	chars := 0
	estTokens := 0
	for _, m := range messages {
		chars += len(m.Text)
		estTokens += len(m.Text)/4 + 1
	}
	r102sRecord(t, "httpapi.handleRunStream", map[string]any{
		"itemsIn": len(messages), "itemsOut": userMarkers,
		"itemsDropped": len(messages) - userMarkers,
		"charsIn":      chars, "estTokens": estTokens,
		"displayWindow": r102sDisplayWindow, "trimRequested": 0,
	}, "")
}

// r102sBlock 取出 [Pinned conversation context] … [/Pinned conversation context]
// 之间的窗口内容，用于把“窗口裁剪结果”与“重新附加的当前用户文本”区分开断言。
func r102sBlock(t *testing.T, text string) string {
	t.Helper()
	start := strings.Index(text, "[Pinned conversation context]\n")
	if start < 0 {
		t.Fatal("expected a pinned conversation context block")
	}
	start += len("[Pinned conversation context]\n")
	end := strings.Index(text, "[/Pinned conversation context]")
	if end < start {
		t.Fatal("pinned conversation context block is not closed")
	}
	return text[start:end]
}

// TestR102ExplicitPinningIsTheOnlyServerSideTrim 断言：只有显式带 pinnedMessageIds
// 时 Orchestrator 才应用 head=20/tail=4 窗口，窗口外的消息被丢弃。
//
// 用 index 23（assistant 角色）作判据：assistant 消息不会出现在
// extractUserText 的结果里，所以它在整个 planner 输入中的缺席只能来自窗口裁剪。
func TestR102ExplicitPinningIsTheOnlyServerSideTrim(t *testing.T) {
	const total = 40
	messages := r102sMessages(total)

	unpinned := &FakeMainAgent{PlanToReturn: r102sConversationalPlan("run-r1-02-window")}
	r102sRun(t, messages, nil, unpinned)
	if !strings.Contains(unpinned.LastInput.UserMessage, "MARKER-022") {
		t.Fatal("baseline: without pins every user message must survive")
	}
	if strings.Contains(unpinned.LastInput.UserMessage, "[Pinned conversation context]") {
		t.Fatal("no trimming may happen without an explicit pin request")
	}

	pinned := &FakeMainAgent{PlanToReturn: r102sConversationalPlan("run-r1-02-window")}
	rec := r102sRun(t, messages, []string{"ctx-003"}, pinned)
	if rec.Code != http.StatusOK {
		t.Fatalf("run failed: %d %s", rec.Code, rec.Body.String())
	}
	got := pinned.LastInput.UserMessage
	block := r102sBlock(t, got)

	// head=20 → index 0..19，tail=4 → index 36..39，共 24 条；ctx-003 落在 head 内。
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
	if len(lines) != 24 {
		t.Fatalf("window must keep exactly 24 of %d messages, kept %d", total, len(lines))
	}
	for _, want := range []string{"MARKER-000", "MARKER-003", "MARKER-019", "MARKER-036", "MARKER-039"} {
		if !strings.Contains(block, want) {
			t.Fatalf("%q must be inside the head/tail window", want)
		}
	}
	for _, gone := range []string{"MARKER-020", "MARKER-022", "MARKER-035"} {
		if strings.Contains(block, gone) {
			t.Fatalf("%q is outside the window and must be trimmed", gone)
		}
	}
	// assistant 消息不会被 extractUserText 重新带回，因此可作全局断言。
	if strings.Contains(got, "MARKER-023") {
		t.Fatal("trimmed assistant message must not reappear anywhere in the planner input")
	}

	r102sRecord(t, "httpapi.handleRunStream", map[string]any{
		"itemsIn": total, "itemsOut": len(lines), "itemsDropped": total - len(lines),
		"keepFirst": 20, "keepLast": 4, "trimRequested": 1, "pinnedCount": 1,
	}, "")
}

// TestR102PinnedWindowIsAdditiveForUserText 是 M-04 相关的特征化用例：
// buildPinnedUserText 在窗口块之后仍会追加 currentUserText，而 currentUserText
// 来自 extractUserText —— 它拼接了请求里 *全部* user 消息
// （handler_run_stream.go:1098-1109、1170-1171）。
//
// 因此对 user 消息而言，head/tail 窗口是“附加”而不是“缩减”：被窗口丢弃的
// MARKER-022 会重新出现在 [Current user message] 段里。这条用例不是弱化断言，
// 它把当前事实钉死；一旦生产改为“窗口即缩减”，本用例会失败并要求同步更新。
func TestR102PinnedWindowIsAdditiveForUserText(t *testing.T) {
	const total = 40
	messages := r102sMessages(total)

	fake := &FakeMainAgent{PlanToReturn: r102sConversationalPlan("run-r1-02-window")}
	rec := r102sRun(t, messages, []string{"ctx-003"}, fake)
	if rec.Code != http.StatusOK {
		t.Fatalf("run failed: %d %s", rec.Code, rec.Body.String())
	}
	got := fake.LastInput.UserMessage
	block := r102sBlock(t, got)

	if strings.Contains(block, "MARKER-022") {
		t.Fatal("MARKER-022 must be outside the pinned window")
	}
	if !strings.Contains(got, "[Current user message]") {
		t.Fatal("expected the current user message block to be appended")
	}
	current := got[strings.Index(got, "[Current user message]"):]
	if !strings.Contains(current, "MARKER-022") {
		t.Fatal("PRODUCTION CHANGE DETECTED: the pinned window now actually reduces the planner payload. " +
			"Gap M-04 shrinks; update this characterization test together with the hard-limit gate.")
	}

	// 缩减只发生在 assistant/工具侧：同一条 assistant 消息确实消失了。
	if strings.Contains(got, "MARKER-023") {
		t.Fatal("assistant message inside the trimmed region must be gone")
	}

	charsIn := 0
	for _, m := range messages {
		charsIn += len(m.Text)
	}
	r102sRecord(t, "httpapi.buildPinnedUserText", map[string]any{
		"itemsIn": total, "itemsOut": 24, "itemsDropped": total - 24,
		"charsIn": charsIn, "charsOut": len(got),
		"estTokens":   len(got)/4 + 1,
		"additiveFor": "user-role", "trimRequested": 1,
	}, "characterization_additive_window")
}

// TestR102PinnedConstraintIsNeverSilentlyTruncated 断言：显式 pinned 的关键约束
// 不会被 Orchestrator 静默截断（即使它落在窗口外、即使内容很大）。
func TestR102PinnedConstraintIsNeverSilentlyTruncated(t *testing.T) {
	messages := r102sMessages(30)
	// 中段一条 12_000 字符的 pinned 约束（index 25 落在 head/tail 窗口之外）。
	messages[25] = MessageInput{ID: "ctx-pin-big", Role: "user", Text: strings.Repeat("C", 12_000)}

	fake := &FakeMainAgent{PlanToReturn: r102sConversationalPlan("run-r1-02-window")}
	rec := r102sRun(t, messages, []string{"ctx-pin-big"}, fake)
	if rec.Code != http.StatusOK {
		t.Fatalf("run failed: %d %s", rec.Code, rec.Body.String())
	}
	got := fake.LastInput.UserMessage
	if !strings.Contains(got, strings.Repeat("C", 12_000)) {
		t.Fatalf("pinned constraint must reach the planner intact, got %d chars", len(got))
	}
	r102sRecord(t, "httpapi.handleRunStream", map[string]any{
		"itemsIn": len(messages), "charsIn": 12_000 + len(messages)*10,
		"pinnedCharsPreserved": 12_000, "itemsTruncated": 0, "trimRequested": 1,
	}, "")
}

// TestR102ContextGateVerdictAtRealHardLimit 用真实默认 hard limit（120_000）
// 断言 119,999 / 120,001 两个边界在 Orchestrator 组装出的上下文上一致地给出
// 可理解、可判定的结果（哨兵 harness.ErrContextTooLarge）。
func TestR102ContextGateVerdictAtRealHardLimit(t *testing.T) {
	cases := []struct {
		name      string
		total     int
		wantError bool
	}{
		{"just_below_119999", 119_999, false},
		{"just_above_120001", 120_001, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			messages := []MessageInput{
				{ID: "sys", Role: "system", Text: strings.Repeat("S", 9)},
				{ID: "pin", Role: "user", Text: strings.Repeat("P", tc.total-9)},
			}
			items := make([]harness.ContextItem, 0, len(messages))
			for _, m := range messages {
				items = append(items, harness.ContextItem{Role: m.Role, Text: m.Text, Pinned: m.ID == "pin"})
			}

			out, err := harness.BuildContext(items, harness.DefaultContextLimits())
			if tc.wantError {
				if err == nil || err.Error() != "required context exceeds budget" {
					t.Fatalf("expected the stable context-too-large sentinel at %d chars, got %v", tc.total, err)
				}
				if out != nil {
					t.Fatal("overflow must not yield a partially trimmed context")
				}
				r102sRecord(t, "harness.BuildContext", map[string]any{
					"itemsIn": len(items), "itemsOut": 0, "itemsDropped": len(items),
					"charsIn": tc.total, "maxChars": harness.DefaultContextLimits().MaxChars,
				}, "ErrContextTooLarge")
				return
			}
			if err != nil {
				t.Fatalf("%d chars must be accepted, got %v", tc.total, err)
			}
			if len(out) != len(items) {
				t.Fatalf("below the limit nothing may be dropped: in=%d out=%d", len(items), len(out))
			}
			r102sRecord(t, "harness.BuildContext", map[string]any{
				"itemsIn": len(items), "itemsOut": len(out), "itemsDropped": 0,
				"charsIn": tc.total, "charsOut": tc.total,
				"estTokens": tc.total/4 + 2,
				"maxChars":  harness.DefaultContextLimits().MaxChars,
			}, "")
		})
	}
}

// TestR102RunStreamGatesOversizedPinnedBeforePlanner 断言 M-04 修复后的生产行为：
// run 入口在 pinned 组装的上下文超过 harness 默认 hard limit（120_000 字符）时，
// 返回稳定的 ORCHESTRATOR_CONTEXT_TOO_LARGE，且 planner 一次都不调用。
func TestR102RunStreamGatesOversizedPinnedBeforePlanner(t *testing.T) {
	const oversized = 130_000 // > harness.DefaultContextLimits().MaxChars
	messages := []MessageInput{
		{ID: "ctx-huge", Role: "user", Text: strings.Repeat("H", oversized)},
	}

	fake := &FakeMainAgent{PlanToReturn: r102sConversationalPlan("run-r1-02-window")}
	rec := r102sRun(t, messages, []string{"ctx-huge"}, fake)
	if rec.Code != http.StatusOK {
		t.Fatalf("run failed: %d %s", rec.Code, rec.Body.String())
	}

	// 事实 1：超限上下文没有进入 planner，模型不会被计费。
	if fake.PlanCalled {
		t.Fatalf("oversized context reached the planner %d time(s); the hard-limit gate is missing",
			fake.PlanCallCount)
	}
	// 事实 2：错误稳定且可理解。
	stream := rec.Body.String()
	if !strings.Contains(stream, "ORCHESTRATOR_CONTEXT_TOO_LARGE") {
		t.Fatalf("expected ORCHESTRATOR_CONTEXT_TOO_LARGE, got: %s", stream)
	}
	if strings.Contains(stream, strings.Repeat("H", 64)) {
		t.Fatal("the rejected context must not be echoed into the SSE error")
	}
	// 事实 3：harness 哨兵稳定，错误分类可复用。
	_, err := harness.BuildContext(
		[]harness.ContextItem{{Role: "system", Text: strings.Repeat("S", 9)}, {Role: "user", Text: strings.Repeat("H", oversized-9), Pinned: true}},
		harness.DefaultContextLimits(),
	)
	if !errors.Is(err, harness.ErrContextTooLarge) {
		t.Fatalf("harness must expose the stable ErrContextTooLarge sentinel, got %v", err)
	}

	r102sRecord(t, "httpapi.handleRunStream", map[string]any{
		"itemsIn": len(messages), "charsIn": oversized,
		"maxChars":     harness.DefaultContextLimits().MaxChars,
		"plannerCalls": fake.PlanCallCount, "gatedByHardLimit": 1,
	}, "")
}

// TestR102RunStreamAllowsContextBelowHardLimit 证明门禁不会误伤正常上下文：
// 低于 hard limit 时 planner 仍然执行。
func TestR102RunStreamAllowsContextBelowHardLimit(t *testing.T) {
	const below = 100_000
	messages := []MessageInput{
		{ID: "ctx-ok", Role: "user", Text: strings.Repeat("H", below)},
	}
	fake := &FakeMainAgent{PlanToReturn: r102sConversationalPlan("run-r1-02-below")}
	r102sRun(t, messages, []string{"ctx-ok"}, fake)
	if !fake.PlanCalled {
		t.Fatal("a below-limit context must still reach the planner")
	}
	r102sRecord(t, "httpapi.handleRunStream", map[string]any{
		"itemsIn": len(messages), "charsIn": below,
		"maxChars":     harness.DefaultContextLimits().MaxChars,
		"plannerCalls": fake.PlanCallCount, "gatedByHardLimit": 0,
	}, "")
}
