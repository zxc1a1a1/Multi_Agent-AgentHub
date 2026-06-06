package planner

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Success cases
// ---------------------------------------------------------------------------

func TestPlanParser_PlainJSON(t *testing.T) {
	p := NewPlanParser()
	raw := `{"intent":"build a web app","mode":"single","confidence":0.95,"steps":[{"agent_name":"code-agent","input":"create the app"}],"user_visible_summary":"I will ask code-agent to build it."}`
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Intent != "build a web app" {
		t.Errorf("expected intent 'build a web app', got %q", schema.Intent)
	}
	if schema.Mode != "single" {
		t.Errorf("expected mode 'single', got %q", schema.Mode)
	}
	if len(schema.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(schema.Steps))
	}
	if schema.Steps[0].AgentName != "code-agent" {
		t.Errorf("expected agent 'code-agent', got %q", schema.Steps[0].AgentName)
	}
}

func TestPlanParser_JSONWithFences(t *testing.T) {
	p := NewPlanParser()
	raw := "```json\n" +
		`{"intent":"test","mode":"parallel","confidence":0.8,"steps":[{"agent_name":"web-agent","input":"build UI"},{"agent_name":"code-agent","input":"build API"}]}` +
		"\n```"
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Mode != "parallel" {
		t.Errorf("expected mode 'parallel', got %q", schema.Mode)
	}
	if len(schema.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(schema.Steps))
	}
}

func TestPlanParser_JSONWithFencesNoLang(t *testing.T) {
	p := NewPlanParser()
	raw := "```\n" +
		`{"intent":"test","mode":"single","steps":[{"agent_name":"code-agent","input":"do it"}]}` +
		"\n```"
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Mode != "single" {
		t.Errorf("expected mode 'single', got %q", schema.Mode)
	}
}

func TestPlanParser_LeadingTrailingWhitespace(t *testing.T) {
	p := NewPlanParser()
	raw := "  \n  " +
		`{"intent":"test","mode":"single","steps":[{"agent_name":"code-agent","input":"do it"}]}` +
		"\n  "
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Mode != "single" {
		t.Errorf("expected mode 'single', got %q", schema.Mode)
	}
}

func TestPlanParser_ExtractFirstJSONFromText(t *testing.T) {
	p := NewPlanParser()
	raw := "I think this plan works best:\n" +
		`{"intent":"test","mode":"single","steps":[{"agent_name":"code-agent","input":"do it"}]}` +
		"\nWhat do you think?"
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Mode != "single" {
		t.Errorf("expected mode 'single', got %q", schema.Mode)
	}
}

func TestPlanParser_MinimumValid(t *testing.T) {
	p := NewPlanParser()
	raw := `{"intent":"simple task","mode":"single","steps":[{"agent_name":"code-agent","input":"do it"}]}`
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Intent != "simple task" {
		t.Errorf("expected intent 'simple task', got %q", schema.Intent)
	}
	if schema.Confidence != 0 {
		t.Errorf("expected confidence 0 for unset, got %f", schema.Confidence)
	}
}

func TestPlanParser_FullSchema(t *testing.T) {
	p := NewPlanParser()
	raw := `{
		"intent": "full stack app",
		"mode": "parallel",
		"confidence": 0.92,
		"steps": [
			{
				"id": "step-1",
				"agent_name": "web-agent",
				"input": "Build a React dashboard",
				"depends_on": [],
				"reason": "User asked for frontend UI"
			},
			{
				"id": "step-2",
				"agent_name": "code-agent",
				"input": "Build the Go API server",
				"depends_on": [],
				"reason": "User asked for backend code"
			}
		],
		"user_visible_summary": "I will build the frontend with web-agent and the backend with code-agent."
	}`
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Intent != "full stack app" {
		t.Errorf("expected intent 'full stack app', got %q", schema.Intent)
	}
	if schema.Mode != "parallel" {
		t.Errorf("expected mode 'parallel', got %q", schema.Mode)
	}
	if schema.Confidence != 0.92 {
		t.Errorf("expected confidence 0.92, got %f", schema.Confidence)
	}
	if len(schema.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(schema.Steps))
	}
	if schema.Steps[0].ID != "step-1" {
		t.Errorf("expected step[0].id 'step-1', got %q", schema.Steps[0].ID)
	}
	if schema.Steps[0].AgentName != "web-agent" {
		t.Errorf("expected step[0].agent_name 'web-agent', got %q", schema.Steps[0].AgentName)
	}
	if schema.Steps[1].ID != "step-2" {
		t.Errorf("expected step[1].id 'step-2', got %q", schema.Steps[1].ID)
	}
	if schema.UserVisibleSummary == "" {
		t.Error("expected non-empty user_visible_summary")
	}
}

// ---------------------------------------------------------------------------
// Rejection cases
// ---------------------------------------------------------------------------

func TestPlanParser_RejectEmpty(t *testing.T) {
	p := NewPlanParser()
	_, err := p.Parse("")
	if err == nil {
		t.Error("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected 'empty' in error, got %v", err)
	}
}

func TestPlanParser_RejectWhitespaceOnly(t *testing.T) {
	p := NewPlanParser()
	_, err := p.Parse("   \n  \t  ")
	if err == nil {
		t.Error("expected error for whitespace-only input")
	}
}

func TestPlanParser_RejectNonJSON(t *testing.T) {
	p := NewPlanParser()
	_, err := p.Parse("this is just some random text with no JSON at all")
	if err == nil {
		t.Error("expected error for non-JSON input")
	}
	if !strings.Contains(err.Error(), "no JSON") {
		t.Errorf("expected 'no JSON' in error, got %v", err)
	}
}

func TestPlanParser_RejectArrayRoot(t *testing.T) {
	p := NewPlanParser()
	_, err := p.Parse(`[{"intent":"test","mode":"single","steps":[{"agent_name":"code-agent","input":"do it"}]}]`)
	if err == nil {
		t.Error("expected error for array root")
	}
	if !strings.Contains(err.Error(), "array") {
		t.Errorf("expected 'array' in error, got %v", err)
	}
}

func TestPlanParser_RejectTruncatedJSON(t *testing.T) {
	// Truncated JSON fails brace-depth matching before json.Unmarshal.
	p := NewPlanParser()
	_, err := p.Parse(`{"intent":"test","mode":"single","steps":[`)
	if err == nil {
		t.Error("expected error for truncated JSON")
	}
	// Both "unclosed" and "invalid JSON" are valid rejection messages.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "unclosed") && !strings.Contains(errMsg, "invalid JSON") {
		t.Errorf("expected rejection for truncated JSON, got %v", err)
	}
}

func TestPlanParser_RejectInvalidJSON(t *testing.T) {
	// Valid JSON structurally but not valid PlanSchema — e.g. random JSON object.
	p := NewPlanParser()
	_, err := p.Parse(`{"foo":1,"bar":"baz"}`)
	if err == nil {
		t.Error("expected error for valid JSON that fails schema validation")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "missing mode") && !strings.Contains(errMsg, "missing steps") {
		t.Errorf("expected 'missing mode' or 'missing steps', got %v", err)
	}
}

func TestPlanParser_RejectMissingMode(t *testing.T) {
	p := NewPlanParser()
	_, err := p.Parse(`{"intent":"test","steps":[{"agent_name":"code-agent","input":"do it"}]}`)
	if err == nil {
		t.Error("expected error for missing mode")
	}
	if !strings.Contains(err.Error(), "missing mode") {
		t.Errorf("expected 'missing mode' in error, got %v", err)
	}
}

func TestPlanParser_RejectModeEmptyString(t *testing.T) {
	p := NewPlanParser()
	_, err := p.Parse(`{"intent":"test","mode":"","steps":[{"agent_name":"code-agent","input":"do it"}]}`)
	if err == nil {
		t.Error("expected error for empty mode string")
	}
	if !strings.Contains(err.Error(), "missing mode") {
		t.Errorf("expected 'missing mode' in error, got %v", err)
	}
}

func TestPlanParser_RejectMissingSteps(t *testing.T) {
	p := NewPlanParser()
	_, err := p.Parse(`{"intent":"test","mode":"single"}`)
	if err == nil {
		t.Error("expected error for missing steps")
	}
	if !strings.Contains(err.Error(), "missing steps") {
		t.Errorf("expected 'missing steps' in error, got %v", err)
	}
}

func TestPlanParser_RejectEmptySteps(t *testing.T) {
	p := NewPlanParser()
	_, err := p.Parse(`{"intent":"test","mode":"single","steps":[]}`)
	if err == nil {
		t.Error("expected error for empty steps array")
	}
	if !strings.Contains(err.Error(), "missing steps") {
		t.Errorf("expected 'missing steps' in error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestPlanParser_NestedBracesInInput(t *testing.T) {
	// The step input may contain JSON-like characters; brace matching must handle them.
	p := NewPlanParser()
	raw := `{"intent":"test","mode":"single","steps":[{"agent_name":"code-agent","input":"Create a handler with {\"key\": \"value\"} payload"}]}`
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(schema.Steps[0].Input, `{"key": "value"}`) {
		t.Errorf("expected nested JSON in input to survive, got %q", schema.Steps[0].Input)
	}
}

func TestPlanParser_UnicodeIntent(t *testing.T) {
	p := NewPlanParser()
	// Chinese characters in intent
	raw := `{"intent":"构建全栈应用","mode":"single","steps":[{"agent_name":"code-agent","input":"生成代码"}]}`
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Intent != "构建全栈应用" {
		t.Errorf("expected Chinese intent, got %q", schema.Intent)
	}
}

func TestPlanParser_ConfidenceOutOfRange(t *testing.T) {
	// The parser does NOT enforce 0..1 range; that is the validator's job.
	p := NewPlanParser()
	raw := `{"intent":"test","mode":"single","confidence":2.5,"steps":[{"agent_name":"code-agent","input":"do it"}]}`
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Confidence != 2.5 {
		t.Errorf("expected confidence 2.5, got %f", schema.Confidence)
	}
}

func TestPlanParser_FencedWithTrailingText(t *testing.T) {
	p := NewPlanParser()
	raw := "```json\n" +
		`{"intent":"test","mode":"single","steps":[{"agent_name":"code-agent","input":"do it"}]}` +
		"\n```\nHope this works!"
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Mode != "single" {
		t.Errorf("expected mode 'single', got %q", schema.Mode)
	}
}

func TestPlanParser_MultipleJSONObjects_PicksFirst(t *testing.T) {
	p := NewPlanParser()
	raw := `{"intent":"first","mode":"single","steps":[{"agent_name":"code-agent","input":"do it"}]}
{"intent":"second","mode":"single","steps":[{"agent_name":"web-agent","input":"do that"}]}`
	schema, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Intent != "first" {
		t.Errorf("expected first JSON object, got %q", schema.Intent)
	}
}
