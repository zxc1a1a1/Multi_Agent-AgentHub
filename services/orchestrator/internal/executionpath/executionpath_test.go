package executionpath

import (
	"testing"
)

// ---- DeriveExecutionPath tests ----

func TestDeriveExecutionPath(t *testing.T) {
	tests := []struct {
		name              string
		requestedPath     string
		agentName         string
		selectedAgentNames []string
		mentions          []string
		wantPath          ChatExecutionPath
		wantErr           bool
		errCode           string
	}{
		// A. Explicit auto
		{
			name:      "agentName=auto → main_agent_orchestration",
			agentName: "auto",
			wantPath:  PathMainAgentOrchestration,
		},
		{
			name:          "requestedPath=auto → main_agent_orchestration",
			requestedPath: "auto",
			wantPath:      PathMainAgentOrchestration,
		},
		{
			name:          "requestedPath=main_agent_orchestration",
			requestedPath: "main_agent_orchestration",
			wantPath:      PathMainAgentOrchestration,
		},

		// B. User did not specify any agent
		{
			name:     "empty everything → main_agent_orchestration",
			wantPath: PathMainAgentOrchestration,
		},

		// C. single_chat
		{
			name:              "selectedAgentNames=[code-agent] → single_chat",
			selectedAgentNames: []string{"code-agent"},
			wantPath:          PathSingleChat,
		},
		{
			name:      "agentName=code-agent → single_chat",
			agentName: "code-agent",
			wantPath:  PathSingleChat,
		},
		{
			name:     "mentions=[code-agent] → single_chat",
			mentions: []string{"code-agent"},
			wantPath: PathSingleChat,
		},

		// D. group_chat
		{
			name:              "selectedAgentNames=[code-agent,review-agent] → group_chat",
			selectedAgentNames: []string{"code-agent", "review-agent"},
			wantPath:          PathGroupChat,
		},
		{
			name:     "mentions=[code-agent,review-agent] → group_chat",
			mentions: []string{"code-agent", "review-agent"},
			wantPath: PathGroupChat,
		},
		{
			name:              "selectedAgentNames + mentions both non-empty → group_chat",
			selectedAgentNames: []string{"code-agent"},
			mentions:          []string{"review-agent"},
			wantPath:          PathGroupChat,
		},

		// E. requestedPath explicit validation
		{
			name:              "requestedPath=group_chat + selectedAgentNames=[code-agent] → group_chat",
			requestedPath:     "group_chat",
			selectedAgentNames: []string{"code-agent"},
			wantPath:          PathGroupChat,
		},
		{
			name:              "requestedPath=single_chat + selectedAgentNames=[code-agent,review-agent] → error",
			requestedPath:     "single_chat",
			selectedAgentNames: []string{"code-agent", "review-agent"},
			wantErr:           true,
			errCode:           ErrCodeInvalidExecutionPath,
		},
		{
			name:          "requestedPath=single_chat + mentions multiple → error",
			requestedPath: "single_chat",
			mentions:      []string{"code-agent", "review-agent"},
			wantErr:       true,
			errCode:       ErrCodeInvalidExecutionPath,
		},
		{
			name:          "requestedPath=auto + selectedAgentNames → main_agent_orchestration (ignored)",
			requestedPath: "main_agent_orchestration",
			selectedAgentNames: []string{"code-agent"},
			wantPath:      PathMainAgentOrchestration,
		},

		// F. Invalid requestedPath → INVALID_EXECUTION_PATH
		{
			name:          "requestedPath=abc → INVALID_EXECUTION_PATH",
			requestedPath: "abc",
			wantErr:       true,
			errCode:       ErrCodeInvalidExecutionPath,
		},
		{
			name:          "requestedPath=invalid_value → INVALID_EXECUTION_PATH",
			requestedPath: "invalid_value",
			wantErr:       true,
			errCode:       ErrCodeInvalidExecutionPath,
		},

		// Edge cases
		{
			name:      "agentName empty, selectedAgentNames empty, mentions empty → auto",
			wantPath:  PathMainAgentOrchestration,
		},
		{
			name:              "selectedAgentNames single + mentions empty → single_chat",
			selectedAgentNames: []string{"web-agent"},
			wantPath:          PathSingleChat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeriveExecutionPath(tt.requestedPath, tt.agentName, tt.selectedAgentNames, tt.mentions)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error (code=%s), got nil", tt.errCode)
				}
				if tt.errCode != "" {
					se, ok := err.(*SelectionError)
					if !ok {
						t.Fatalf("expected *SelectionError, got %T: %v", err, err)
					}
					if se.Code != tt.errCode {
						t.Fatalf("expected error code %q, got %q", tt.errCode, se.Code)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantPath {
				t.Fatalf("DeriveExecutionPath = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

// ---- NormalizeAgentNames tests ----

func TestNormalizeAgentNames(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "strips @ prefix",
			input: []string{"@code-agent", "@review-agent"},
			want:  []string{"code-agent", "review-agent"},
		},
		{
			name:  "filters empty strings",
			input: []string{"", "@code-agent", " ", "@review-agent"},
			want:  []string{"code-agent", "review-agent"},
		},
		{
			name:  "deduplicates",
			input: []string{"code-agent", "code-agent", "review-agent"},
			want:  []string{"code-agent", "review-agent"},
		},
		{
			name:  "filters auto",
			input: []string{"auto", "code-agent", "auto"},
			want:  []string{"code-agent"},
		},
		{
			name:  "mixed @ and bare",
			input: []string{"@code-agent", "code-agent", "@web-agent"},
			want:  []string{"code-agent", "web-agent"},
		},
		{
			name:  "nil input",
			input: nil,
			want:  []string{},
		},
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "only auto",
			input: []string{"auto", "@auto"},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeAgentNames(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("NormalizeAgentNames = %v (len=%d), want %v (len=%d)", got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("NormalizeAgentNames[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ---- ValidateAgentSelection tests ----

var testAvailableAgents = []string{"code-agent", "web-agent", "review-agent", "document-agent", "test-agent"}

func TestValidateAgentSelection(t *testing.T) {
	tests := []struct {
		name              string
		execPath          ChatExecutionPath
		agentName         string
		selectedAgentNames []string
		mentions          []string
		availableAgents   []string
		wantAllowed       []string
		wantErr           bool
		errCode           string
	}{
		// single_chat
		{
			name:              "single_chat: selectedAgentNames=[code-agent] → allowedAgents=[code-agent]",
			execPath:          PathSingleChat,
			selectedAgentNames: []string{"code-agent"},
			availableAgents:   testAvailableAgents,
			wantAllowed:       []string{"code-agent"},
		},
		{
			name:              "single_chat: mentions=[web-agent] → allowedAgents=[web-agent]",
			execPath:          PathSingleChat,
			mentions:          []string{"@web-agent"},
			availableAgents:   testAvailableAgents,
			wantAllowed:       []string{"web-agent"},
		},
		{
			name:            "single_chat: agentName=code-agent → allowedAgents=[code-agent]",
			execPath:        PathSingleChat,
			agentName:       "code-agent",
			availableAgents: testAvailableAgents,
			wantAllowed:     []string{"code-agent"},
		},
		{
			name:            "single_chat: agentName=auto, no selected/mentions → INVALID_AGENT_SELECTION",
			execPath:        PathSingleChat,
			agentName:       "auto",
			availableAgents: testAvailableAgents,
			wantErr:         true,
			errCode:         ErrCodeInvalidAgentSelection,
			},
			{
			name:            "single_chat: agentName=nonexistent-agent → UNKNOWN_AGENT",
			execPath:        PathSingleChat,
			agentName:       "nonexistent-agent",
			availableAgents: testAvailableAgents,
			wantErr:         true,
			errCode:         ErrCodeUnknownAgent,
		},

		// group_chat
		{
			name:              "group_chat: selectedAgentNames=[code-agent,review-agent], mentions=[] → allowedAgents=[code-agent,review-agent]",
			execPath:          PathGroupChat,
			selectedAgentNames: []string{"code-agent", "review-agent"},
			availableAgents:   testAvailableAgents,
			wantAllowed:       []string{"code-agent", "review-agent"},
		},
		{
			name:              "group_chat: selectedAgentNames=[code-agent,review-agent], mentions=[review-agent] → intersection=[review-agent]",
			execPath:          PathGroupChat,
			selectedAgentNames: []string{"code-agent", "review-agent"},
			mentions:          []string{"review-agent"},
			availableAgents:   testAvailableAgents,
			wantAllowed:       []string{"review-agent"},
		},
		{
			name:              "group_chat: selectedAgentNames=[code-agent], mentions=[web-agent] → AGENT_SELECTION_CONFLICT",
			execPath:          PathGroupChat,
			selectedAgentNames: []string{"code-agent"},
			mentions:          []string{"web-agent"},
			availableAgents:   testAvailableAgents,
			wantErr:           true,
			errCode:           ErrCodeAgentSelectionConflict,
		},
		{
			name:              "group_chat: mentions only → allowedAgents=mentions",
			execPath:          PathGroupChat,
			mentions:          []string{"@code-agent", "@web-agent"},
			availableAgents:   testAvailableAgents,
			wantAllowed:       []string{"code-agent", "web-agent"},
		},

		// main_agent_orchestration
		{
			name:            "auto: all enabled agents allowed",
			execPath:        PathMainAgentOrchestration,
			availableAgents: testAvailableAgents,
			wantAllowed:     []string{"code-agent", "document-agent", "review-agent", "test-agent", "web-agent"}, // sorted
		},
		{
			name:            "auto: with agentName=auto → all enabled agents allowed (not an error)",
			execPath:        PathMainAgentOrchestration,
			agentName:       "auto",
			availableAgents: testAvailableAgents,
			wantAllowed:     []string{"code-agent", "document-agent", "review-agent", "test-agent", "web-agent"},
		},
			{
				name:              "auto: selectedAgentNames=[nonexistent-agent] → pass, all enabled agents allowed",
				execPath:          PathMainAgentOrchestration,
				selectedAgentNames: []string{"nonexistent-agent"},
				availableAgents:   testAvailableAgents,
				wantAllowed:       []string{"code-agent", "document-agent", "review-agent", "test-agent", "web-agent"},
			},
			{
				name:            "auto: mentions=[@nonexistent-agent] → pass, all enabled agents allowed",
				execPath:        PathMainAgentOrchestration,
				mentions:        []string{"@nonexistent-agent"},
				availableAgents: testAvailableAgents,
				wantAllowed:     []string{"code-agent", "document-agent", "review-agent", "test-agent", "web-agent"},
			},


		// Unknown agent
		{
			name:              "unknown agent → UNKNOWN_AGENT",
			execPath:          PathSingleChat,
			selectedAgentNames: []string{"nonexistent-agent"},
			availableAgents:   testAvailableAgents,
			wantErr:           true,
			errCode:           ErrCodeUnknownAgent,
		},
		{
			name:            "unknown agent in mentions → UNKNOWN_AGENT",
			execPath:        PathGroupChat,
			mentions:        []string{"@nonexistent"},
			availableAgents: testAvailableAgents,
			wantErr:         true,
			errCode:         ErrCodeUnknownAgent,
		},
		{
			name:            "agentName=nonexistent → UNKNOWN_AGENT",
			execPath:        PathSingleChat,
			agentName:       "nonexistent-agent",
			availableAgents: testAvailableAgents,
			wantErr:         true,
			errCode:         ErrCodeUnknownAgent,
		},

		// Invalid execution path
		{
			name:     "invalid execution path",
			execPath: ChatExecutionPath("invalid"),
			wantErr:  true,
			errCode:  ErrCodeInvalidExecutionPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateAgentSelection(tt.execPath, tt.agentName, tt.selectedAgentNames, tt.mentions, tt.availableAgents)
			if tt.wantErr {
				if result.Error == nil {
					t.Fatalf("expected error (code=%s), got nil", tt.errCode)
				}
				if tt.errCode != "" && result.Error.Code != tt.errCode {
					t.Fatalf("expected error code %q, got %q", tt.errCode, result.Error.Code)
				}
				return
			}
			if result.Error != nil {
				t.Fatalf("unexpected error: %v", result.Error)
			}
			if !stringSlicesEqual(result.AllowedAgents, tt.wantAllowed) {
				t.Fatalf("AllowedAgents = %v, want %v", result.AllowedAgents, tt.wantAllowed)
			}
		})
	}
}

// ---- EnforceAgentBoundary tests ----

func TestEnforceAgentBoundary(t *testing.T) {
	tasks := func(agents ...string) []TaskInfo {
		out := make([]TaskInfo, len(agents))
		for i, a := range agents {
			out[i] = TaskInfo{TaskID: "task-" + string(rune('1'+i)), AgentName: a}
		}
		return out
	}

	tests := []struct {
		name          string
		tasks         []TaskInfo
		allowedAgents []string
		wantErr       bool
		errCode       string
	}{
		{
			name:          "single_chat: agent in bounds → pass",
			tasks:         tasks("code-agent"),
			allowedAgents: []string{"code-agent"},
		},
		{
			name:          "single_chat: agent out of bounds → AGENT_BOUNDARY_VIOLATION",
			tasks:         tasks("web-agent"),
			allowedAgents: []string{"code-agent"},
			wantErr:       true,
			errCode:       ErrCodeAgentBoundaryViolation,
		},
		{
			name:          "group_chat: all tasks in bounds → pass",
			tasks:         tasks("code-agent", "review-agent"),
			allowedAgents: []string{"code-agent", "review-agent"},
		},
		{
			name:          "group_chat: one task out of bounds → AGENT_BOUNDARY_VIOLATION",
			tasks:         tasks("code-agent", "test-agent"),
			allowedAgents: []string{"code-agent", "review-agent"},
			wantErr:       true,
			errCode:       ErrCodeAgentBoundaryViolation,
		},
		{
			name:          "auto: in bounds → pass",
			tasks:         tasks("code-agent"),
			allowedAgents: []string{"code-agent", "web-agent", "review-agent"},
		},
		{
			name:          "auto: unknown agent → fail",
			tasks:         tasks("unknown-agent"),
			allowedAgents: []string{"code-agent", "web-agent"},
			wantErr:       true,
			errCode:       ErrCodeAgentBoundaryViolation,
		},
		{
			name:          "empty tasks → pass",
			tasks:         []TaskInfo{},
			allowedAgents: []string{"code-agent"},
		},
		{
			name:          "case insensitive match",
			tasks:         tasks("Code-Agent"),
			allowedAgents: []string{"code-agent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnforceAgentBoundary(tt.tasks, tt.allowedAgents)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error (code=%s), got nil", tt.errCode)
				}
				if tt.errCode != "" && err.Code != tt.errCode {
					t.Fatalf("expected error code %q, got %q", tt.errCode, err.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ---- Helpers ----

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
