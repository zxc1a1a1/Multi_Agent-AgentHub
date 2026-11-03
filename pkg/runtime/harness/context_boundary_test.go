package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// R1-02 短期上下文边界测试（plan §4.2 R1-02 / 缺口 M-04、P-02）。
//
// 覆盖的真实生产限制：
//   - DefaultContextLimits: MaxChars=120_000、KeepFirst=3、KeepLast=20（context.go:28）
//   - 超限时返回稳定哨兵 ErrContextTooLarge（context.go:8、context.go:114）
//   - pinned/system 永不静默丢弃（context.go:46、context.go:89）
//   - 孤立 tool result 丢弃、tool call/result 成对保留（context.go:61、context.go:77）
//
// 证据只记录字符数、估算 token 数与丢弃条数，绝不记录消息正文。

const (
	// r102HardLimit 与生产默认 hard limit 对齐（context.go:29）。
	r102HardLimit = 120_000
	// r102TokenDivisor 是 MemoryController.filterRecall 使用的 token 估算口径
	// （controller.go:134: len(content)/4 + 1），此处复用同一口径以便证据可比。
	r102TokenDivisor = 4
)

// r102Metrics 是一条脱敏后的度量记录，字段全部为数量，没有正文。
type r102Metrics struct {
	Test            string `json:"test"`
	ItemsIn         int    `json:"itemsIn"`
	ItemsOut        int    `json:"itemsOut"`
	ItemsDropped    int    `json:"itemsDropped"`
	CharsIn         int    `json:"charsIn"`
	CharsOut        int    `json:"charsOut"`
	EstTokensOut    int    `json:"estTokensOut"`
	MaxChars        int    `json:"maxChars"`
	KeepFirst       int    `json:"keepFirst"`
	KeepLast        int    `json:"keepLast"`
	ErrorClass      string `json:"errorClass"`
	PinnedPreserved bool   `json:"pinnedPreserved"`
}

// r102Record 把度量写入 $AGENTHUB_EVIDENCE_JSONL（scripts/run-integration-evidence.sh 采集）。
// 未设置该环境变量时（普通 go test）不做任何事。
func r102Record(t *testing.T, m r102Metrics) {
	t.Helper()
	path := os.Getenv("AGENTHUB_EVIDENCE_JSONL")
	if path == "" {
		return
	}
	m.Test = t.Name()
	row := map[string]any{
		"schemaVersion": 2,
		"evidenceId":    fmt.Sprintf("ev_r1_02_%s", t.Name()),
		"testId":        "R1-02",
		"planTaskId":    "R1-02",
		"layer":         "orchestrator",
		"actor":         "orchestrator",
		// context_trim 是 §5.2 action 枚举的显式扩展：该枚举没有上下文窗口动词，
		// 已在 R1-02 summary.md 中说明。counts/detail 只使用数量与白名单键。
		"action":    "context_trim",
		"target":    "harness.BuildContext",
		"status":    "ok",
		"errorCode": m.ErrorClass,
		"counts": map[string]any{
			"itemsIn":      m.ItemsIn,
			"itemsOut":     m.ItemsOut,
			"itemsDropped": m.ItemsDropped,
			"charsIn":      m.CharsIn,
			"charsOut":     m.CharsOut,
			"estTokens":    m.EstTokensOut,
			"maxChars":     m.MaxChars,
		},
		"detail": map[string]any{
			"errorClass": m.ErrorClass,
		},
		"recordedAt": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	if m.ErrorClass != "" {
		row["status"] = "error"
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Logf("evidence append failed: %v", err)
		return
	}
	defer file.Close()
	encoded, err := json.Marshal(row)
	if err != nil {
		return
	}
	_, _ = file.Write(append(encoded, '\n'))
}

func r102Chars(items []ContextItem) int {
	total := 0
	for _, item := range items {
		total += len(item.Text)
	}
	return total
}

func r102EstTokens(items []ContextItem) int {
	total := 0
	for _, item := range items {
		total += len(item.Text)/r102TokenDivisor + 1
	}
	return total
}

func r102Texts(items []ContextItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Text)
	}
	return out
}

func r102IndexOf(items []ContextItem, text string) int {
	for i, item := range items {
		if item.Text == text {
			return i
		}
	}
	return -1
}

// r102PinnedOnly 构造 totalChars 个字符的 pinned+system 语料：这类条目在
// BuildContext 中不可被裁剪，因此总字符数就是与 hard limit 比较的量。
func r102PinnedOnly(total int) []ContextItem {
	if total < 10 {
		panic("total must exceed the marker overhead")
	}
	system := 10
	pinned := total - system
	return []ContextItem{
		{Role: "system", Text: strings.Repeat("s", system)},
		{Role: "user", Text: strings.Repeat("p", pinned), Pinned: true},
	}
}

func TestR102DefaultContextLimitsMatchContract(t *testing.T) {
	limits := DefaultContextLimits()
	if limits.MaxChars != r102HardLimit {
		t.Fatalf("MaxChars drifted: got %d want %d", limits.MaxChars, r102HardLimit)
	}
	if limits.KeepFirst != 3 || limits.KeepLast != 20 {
		t.Fatalf("window drifted: got first=%d last=%d want 3/20", limits.KeepFirst, limits.KeepLast)
	}
}

// TestR102BoundaryJustBelowAtAndJustAboveHardLimit 是 P-02 的核心用例：
// 119,999 / 120,000 / 120,001 三个点必须给出确定性、可理解的判定。
func TestR102BoundaryJustBelowAtAndJustAboveHardLimit(t *testing.T) {
	cases := []struct {
		name      string
		total     int
		wantError bool
	}{
		{"just_below_119999", r102HardLimit - 1, false},
		{"exactly_at_120000", r102HardLimit, false},
		{"just_above_120001", r102HardLimit + 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := r102PinnedOnly(tc.total)
			out, err := BuildContext(items, DefaultContextLimits())

			inChars := r102Chars(items)
			if inChars != tc.total {
				t.Fatalf("corpus size wrong: got %d want %d", inChars, tc.total)
			}

			if tc.wantError {
				if !errors.Is(err, ErrContextTooLarge) {
					t.Fatalf("expected ErrContextTooLarge at %d chars, got %v", tc.total, err)
				}
				if err.Error() != "required context exceeds budget" {
					t.Fatalf("error text is not stable: %q", err.Error())
				}
				if out != nil {
					t.Fatalf("overflow must not return a silently trimmed context, got %d items", len(out))
				}
				r102Record(t, r102Metrics{
					ItemsIn: len(items), ItemsOut: 0, ItemsDropped: len(items),
					CharsIn: inChars, CharsOut: 0, EstTokensOut: 0,
					MaxChars: r102HardLimit, KeepFirst: 3, KeepLast: 20,
					ErrorClass: "ErrContextTooLarge", PinnedPreserved: false,
				})
				return
			}

			if err != nil {
				t.Fatalf("context of %d chars must be accepted, got %v", tc.total, err)
			}
			// 边界内不得丢弃任何 pinned/system 约束。
			if len(out) != len(items) {
				t.Fatalf("nothing may be dropped below the limit: in=%d out=%d", len(items), len(out))
			}
			if r102Chars(out) != tc.total {
				t.Fatalf("chars must be preserved exactly: got %d want %d", r102Chars(out), tc.total)
			}
			r102Record(t, r102Metrics{
				ItemsIn: len(items), ItemsOut: len(out), ItemsDropped: len(items) - len(out),
				CharsIn: inChars, CharsOut: r102Chars(out), EstTokensOut: r102EstTokens(out),
				MaxChars: r102HardLimit, KeepFirst: 3, KeepLast: 20,
				ErrorClass: "", PinnedPreserved: true,
			})
		})
	}
}

// TestR102PinnedOverflowNeverSilentlyDropsConstraints 断言 M-04 要求的行为：
// 超限时明确失败，而不是丢掉某条 pinned 约束后“看起来成功”。
func TestR102PinnedOverflowNeverSilentlyDropsConstraints(t *testing.T) {
	items := []ContextItem{
		{Role: "system", Text: strings.Repeat("s", 60_000)},
		{Role: "user", Text: strings.Repeat("a", 40_000), Pinned: true},
		{Role: "user", Text: strings.Repeat("b", 30_000), Pinned: true},
		{Role: "user", Text: "filler", Pinned: false},
	}
	limits := ContextLimits{MaxChars: 100_000, KeepFirst: 1, KeepLast: 1}

	first, firstErr := BuildContext(items, limits)
	second, secondErr := BuildContext(items, limits)

	if !errors.Is(firstErr, ErrContextTooLarge) || !errors.Is(secondErr, ErrContextTooLarge) {
		t.Fatalf("both calls must fail with the sentinel: %v / %v", firstErr, secondErr)
	}
	if firstErr.Error() != secondErr.Error() {
		t.Fatalf("error must be deterministic: %q vs %q", firstErr.Error(), secondErr.Error())
	}
	if first != nil || second != nil {
		t.Fatal("overflow must return no items at all")
	}

	// 只要 pinned 总量本身超过上限，任何非 pinned 的裁剪都无法挽救：
	// 把可裁剪条目全部清空后依然是同一个哨兵错误。
	pinnedOnly := []ContextItem{
		{Role: "system", Text: strings.Repeat("s", 60_000)},
		{Role: "user", Text: strings.Repeat("a", 40_000), Pinned: true},
		{Role: "user", Text: strings.Repeat("b", 30_000), Pinned: true},
	}
	_, err := BuildContext(pinnedOnly, limits)
	if !errors.Is(err, ErrContextTooLarge) {
		t.Fatalf("pinned-only overflow must be the same sentinel, got %v", err)
	}
	r102Record(t, r102Metrics{
		ItemsIn: len(items), ItemsOut: 0, ItemsDropped: len(items),
		CharsIn: r102Chars(items), MaxChars: limits.MaxChars,
		KeepFirst: limits.KeepFirst, KeepLast: limits.KeepLast,
		ErrorClass: "ErrContextTooLarge", PinnedPreserved: false,
	})
}

// TestR102SystemAndPinnedSurviveTrimming 证明裁剪只作用于中段可裁剪条目。
func TestR102SystemAndPinnedSurviveTrimming(t *testing.T) {
	items := []ContextItem{
		{Role: "system", Text: "SYSTEM-CONSTRAINT-0000"},
		{Role: "user", Text: "PINNED-CONSTRAINT-0001", Pinned: true},
	}
	for i := 0; i < 20; i++ {
		items = append(items, ContextItem{Role: "user", Text: fmt.Sprintf("noise-%02d-%s", i, strings.Repeat("n", 40))})
	}
	items = append(items, ContextItem{Role: "user", Text: "PINNED-LATE-CONSTRAINT", Pinned: true})

	limits := ContextLimits{MaxChars: 200, KeepFirst: 1, KeepLast: 2}
	out, err := BuildContext(items, limits)
	if err != nil {
		t.Fatalf("trimming should succeed: %v", err)
	}
	if r102IndexOf(out, "SYSTEM-CONSTRAINT-0000") < 0 {
		t.Fatal("system item must be preserved")
	}
	if r102IndexOf(out, "PINNED-CONSTRAINT-0001") < 0 {
		t.Fatal("early pinned item must be preserved")
	}
	if r102IndexOf(out, "PINNED-LATE-CONSTRAINT") < 0 {
		t.Fatal("late pinned item must be preserved")
	}
	kept := map[string]bool{}
	for _, text := range r102Texts(out) {
		kept[text] = true
	}
	dropped := 0
	for _, item := range items {
		if !kept[item.Text] {
			dropped++
			if item.Pinned || item.Role == "system" {
				t.Fatalf("pinned/system item was silently dropped: %q", item.Text)
			}
			if !strings.HasPrefix(item.Text, "noise-") {
				t.Fatalf("only noise may be dropped, got %q", item.Text)
			}
		}
	}
	if dropped == 0 {
		t.Fatal("nothing was trimmed, the corpus does not exercise the limit")
	}
	if out[0].Role != "system" || out[1].Text != "PINNED-CONSTRAINT-0001" {
		t.Fatalf("original order must be preserved, got %#v", out[:2])
	}
	r102Record(t, r102Metrics{
		ItemsIn: len(items), ItemsOut: len(out), ItemsDropped: dropped,
		CharsIn: r102Chars(items), CharsOut: r102Chars(out), EstTokensOut: r102EstTokens(out),
		MaxChars: limits.MaxChars, KeepFirst: limits.KeepFirst, KeepLast: limits.KeepLast,
		PinnedPreserved: true,
	})
}

// TestR102ToolCallResultPairsAreNeverSplit 扫描所有 keepLast 取值，断言
// 输出里永远不会出现“半个工具调用对”。这是真正会“咬人”的用例：
// 只要成对逻辑被破坏，某个窗口取值必然失败。
func TestR102ToolCallResultPairsAreNeverSplit(t *testing.T) {
	build := func() []ContextItem {
		items := []ContextItem{{Role: "system", Text: "sys"}}
		for i := 0; i < 8; i++ {
			id := fmt.Sprintf("call-%d", i)
			items = append(items,
				ContextItem{Role: "user", Text: fmt.Sprintf("turn-%d", i)},
				ContextItem{Role: "assistant", Text: fmt.Sprintf("call-%d", i), ToolCallID: id, ToolName: "workspace.read"},
				ContextItem{Role: "tool", Text: fmt.Sprintf("result-%d", i), ToolCallID: id, ToolName: "workspace.read", ToolResult: true},
			)
		}
		return items
	}

	for keepLast := 0; keepLast <= 12; keepLast++ {
		items := build()
		out, err := BuildContext(items, ContextLimits{MaxChars: 10_000, KeepFirst: 1, KeepLast: keepLast})
		if err != nil {
			t.Fatalf("keepLast=%d unexpected error: %v", keepLast, err)
		}

		calls := map[string]bool{}
		results := map[string]bool{}
		for _, item := range out {
			if item.ToolCallID == "" {
				continue
			}
			if item.ToolResult {
				results[item.ToolCallID] = true
			} else {
				calls[item.ToolCallID] = true
			}
		}
		for id := range calls {
			if !results[id] {
				t.Fatalf("keepLast=%d kept tool call %s without its result", keepLast, id)
			}
		}
		for id := range results {
			if !calls[id] {
				t.Fatalf("keepLast=%d kept orphan tool result %s", keepLast, id)
			}
		}
		for i := 1; i < len(out); i++ {
			if out[i].Role == "system" {
				t.Fatalf("system item must stay at index 0, found at %d", i)
			}
		}
	}
	r102Record(t, r102Metrics{
		ItemsIn: len(build()), ItemsOut: 0, ItemsDropped: 0,
		CharsIn: r102Chars(build()), MaxChars: 10_000, KeepFirst: 1, KeepLast: 12,
	})
}

// TestR102OrphanToolResultDroppedInsideWindow 断言：即使孤立 tool result 落在
// head/tail 窗口里，也必须被丢弃而不是伪造一个 call。
func TestR102OrphanToolResultDroppedInsideWindow(t *testing.T) {
	items := []ContextItem{
		{Role: "system", Text: "sys"},
		{Role: "user", Text: "u1"},
		{Role: "tool", Text: "ORPHAN-RESULT", ToolCallID: "never-called", ToolName: "apply_patch", ToolResult: true},
		{Role: "user", Text: "u2"},
	}
	out, err := BuildContext(items, ContextLimits{MaxChars: 1_000, KeepFirst: 1, KeepLast: 4})
	if err != nil {
		t.Fatal(err)
	}
	if r102IndexOf(out, "ORPHAN-RESULT") >= 0 {
		t.Fatal("orphan tool result inside the window must still be dropped")
	}
	for _, item := range out {
		if item.ToolResult && item.ToolCallID == "never-called" {
			t.Fatal("orphan result leaked back into the context")
		}
	}
	for _, item := range out {
		if item.ToolCallID != "" && !item.ToolResult {
			t.Fatal("BuildContext must not fabricate a tool call to pair an orphan result")
		}
	}
	r102Record(t, r102Metrics{
		ItemsIn: len(items), ItemsOut: len(out), ItemsDropped: len(items) - len(out),
		CharsIn: r102Chars(items), CharsOut: r102Chars(out), EstTokensOut: r102EstTokens(out),
		MaxChars: 1_000, KeepFirst: 1, KeepLast: 4, PinnedPreserved: true,
	})
}

// TestR102HeadTailWindowSelectionIsDeterministic 断言同样的输入永远得到
// 同样的窗口，且窗口就是 head/tail 语义。
func TestR102HeadTailWindowSelectionIsDeterministic(t *testing.T) {
	items := []ContextItem{{Role: "system", Text: "sys"}}
	for i := 0; i < 30; i++ {
		items = append(items, ContextItem{Role: "user", Text: fmt.Sprintf("msg-%02d", i)})
	}
	limits := ContextLimits{MaxChars: 10_000, KeepFirst: 4, KeepLast: 6}

	first, err := BuildContext(items, limits)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildContext(items, limits)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(r102Texts(first), "|") != strings.Join(r102Texts(second), "|") {
		t.Fatal("window selection is not deterministic")
	}

	want := map[string]bool{"sys": true}
	// KeepFirst 覆盖 index 0..KeepFirst-1，index 0 是 system 自身。
	for i := 1; i < limits.KeepFirst; i++ {
		want[fmt.Sprintf("msg-%02d", i-1)] = true
	}
	for i := 30 - 6; i < 30; i++ {
		want[fmt.Sprintf("msg-%02d", i)] = true
	}
	got := map[string]bool{}
	for _, item := range first {
		got[item.Text] = true
	}
	if len(got) != len(want) {
		t.Fatalf("window size mismatch: got %d want %d (%v)", len(got), len(want), got)
	}
	for text := range want {
		if !got[text] {
			t.Fatalf("expected %q inside the head/tail window", text)
		}
	}
	r102Record(t, r102Metrics{
		ItemsIn: len(items), ItemsOut: len(first), ItemsDropped: len(items) - len(first),
		CharsIn: r102Chars(items), CharsOut: r102Chars(first), EstTokensOut: r102EstTokens(first),
		MaxChars: limits.MaxChars, KeepFirst: limits.KeepFirst, KeepLast: limits.KeepLast,
		PinnedPreserved: true,
	})
}

// TestR102TrimmingStopsAtTheLimitInsteadOfGoingOver 断言裁剪循环收敛后
// 结果字符数必然 <= MaxChars（哨兵只在无法收敛时出现）。
func TestR102TrimmingStopsAtTheLimitInsteadOfGoingOver(t *testing.T) {
	for _, maxChars := range []int{40, 80, 120, 200, 400} {
		items := []ContextItem{{Role: "system", Text: strings.Repeat("s", 10)}}
		for i := 0; i < 30; i++ {
			items = append(items, ContextItem{Role: "user", Text: strings.Repeat("x", 15)})
		}
		out, err := BuildContext(items, ContextLimits{MaxChars: maxChars, KeepFirst: 1, KeepLast: 5})
		if err != nil {
			if !errors.Is(err, ErrContextTooLarge) {
				t.Fatalf("maxChars=%d unexpected error %v", maxChars, err)
			}
			continue
		}
		if r102Chars(out) > maxChars {
			t.Fatalf("maxChars=%d returned %d chars over the limit", maxChars, r102Chars(out))
		}
	}
}

func TestR102BuildContextRejectsInvalidLimits(t *testing.T) {
	items := []ContextItem{{Role: "user", Text: "hello"}}
	if _, err := BuildContext(items, ContextLimits{MaxChars: 0}); err == nil {
		t.Fatal("MaxChars<=0 must be rejected")
	}
	if _, err := BuildContext(items, ContextLimits{MaxChars: -5}); err == nil {
		t.Fatal("negative MaxChars must be rejected")
	}
	if _, err := BuildContext(items, ContextLimits{MaxChars: 10, KeepFirst: -1}); err == nil {
		t.Fatal("negative KeepFirst must be rejected")
	}
	if _, err := BuildContext(items, ContextLimits{MaxChars: 10, KeepLast: -1}); err == nil {
		t.Fatal("negative KeepLast must be rejected")
	}
}
