package adk

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"time"
)

const defaultMaxIterations = 25

// RunOption customizes Runner behavior.
type RunOption func(*Runner)

// WithMaxIterations overrides the default max loop count.
func WithMaxIterations(n int) RunOption {
	return func(r *Runner) {
		if r == nil || n <= 0 {
			return
		}
		r.maxIterations = n
	}
}

// WithTools sets the tool set available to model generation requests.
func WithTools(tools ...Tool) RunOption {
	return func(r *Runner) {
		if r == nil || len(tools) == 0 {
			return
		}
		r.tools = append(r.tools, tools...)
	}
}

// WithPlugins sets lifecycle plugins for generation and tool execution.
func WithPlugins(plugins ...Plugin) RunOption {
	return func(r *Runner) {
		if r == nil || len(plugins) == 0 {
			return
		}
		r.plugins = append(r.plugins, plugins...)
	}
}

// Runner drives multi-turn Generate -> ToolCall -> ToolResult loops.
type Runner struct {
	agent         Agent
	session       SessionService
	tools         []Tool
	plugins       []Plugin
	maxIterations int
}

// NewRunner creates a runner with default options.
func NewRunner(agent Agent, session SessionService, opts ...RunOption) *Runner {
	runner := &Runner{
		agent:         agent,
		session:       session,
		tools:         make([]Tool, 0),
		plugins:       make([]Plugin, 0),
		maxIterations: defaultMaxIterations,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(runner)
	}

	if runner.maxIterations <= 0 {
		runner.maxIterations = defaultMaxIterations
	}

	return runner
}

// Run executes one user turn and yields runtime events.
func (r *Runner) Run(ctx context.Context, sessionID string, userContent *Content) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		if r == nil {
			yield(Event{}, errors.New("runner is nil"))
			return
		}
		if r.agent == nil {
			yield(Event{}, errors.New("runner agent is nil"))
			return
		}
		if r.session == nil {
			yield(Event{}, errors.New("runner session service is nil"))
			return
		}
		if ctx == nil {
			ctx = context.Background()
		}
		if err := ctx.Err(); err != nil {
			yield(Event{}, err)
			return
		}

		sess, err := r.session.Get(ctx, sessionID)
		if err != nil {
			yield(Event{}, err)
			return
		}
		if sess == nil {
			yield(Event{}, errors.New("session is nil"))
			return
		}
		if sess.State == nil {
			sess.State = NewSessionState(nil)
		}

		userEvent := Event{
			Author:    "user",
			Content:   userContent,
			Timestamp: time.Now(),
		}
		if err := r.session.AppendEvent(ctx, sessionID, userEvent); err != nil {
			yield(Event{}, err)
			return
		}

		contents := buildContents(sess, userContent)

		for iteration := 0; iteration < r.maxIterations; iteration++ {
			if err := ctx.Err(); err != nil {
				yield(Event{}, err)
				return
			}

			req := &GenerateRequest{
				Contents: contents,
				Tools:    append([]Tool(nil), r.tools...),
				Config:   nil,
			}

			for _, plugin := range r.plugins {
				if plugin == nil {
					continue
				}
				if err := plugin.BeforeGenerate(ctx, sess.State, req); err != nil {
					yield(Event{}, err)
					return
				}
			}

			resp, err := r.agent.Generate(ctx, req)
			if err != nil {
				yield(Event{}, err)
				return
			}
			if resp == nil {
				yield(Event{}, errors.New("agent returned nil response"))
				return
			}

			for _, plugin := range r.plugins {
				if plugin == nil {
					continue
				}
				if err := plugin.AfterGenerate(ctx, sess.State, resp); err != nil {
					yield(Event{}, err)
					return
				}
			}

			assistantContent := &Content{
				Role:  RoleAssistant,
				Parts: append([]Part(nil), resp.Parts...),
			}
			assistantEvent := Event{
				Author:    r.agent.Name(),
				Content:   assistantContent,
				Timestamp: time.Now(),
			}

			if resp.FinishReason == FinishToolUse {
				if !yield(assistantEvent, nil) {
					return
				}

				toolResults, err := r.executeTools(ctx, sess, resp.Parts)
				if err != nil {
					yield(Event{}, err)
					return
				}

				toolParts := make([]Part, len(toolResults))
				for i := range toolResults {
					toolParts[i] = toolResults[i]
				}

				toolResultContent := &Content{
					Role:  RoleTool,
					Parts: toolParts,
				}
				toolResultEvent := Event{
					Author:    r.agent.Name(),
					Content:   toolResultContent,
					Timestamp: time.Now(),
				}

				if !yield(toolResultEvent, nil) {
					return
				}

				if err := r.session.AppendEvent(ctx, sessionID, assistantEvent); err != nil {
					yield(Event{}, err)
					return
				}
				if err := r.session.AppendEvent(ctx, sessionID, toolResultEvent); err != nil {
					yield(Event{}, err)
					return
				}

				contents = append(contents, assistantContent, toolResultContent)
				continue
			}

			assistantEvent.Final = true
			if !yield(assistantEvent, nil) {
				return
			}
			if err := r.session.AppendEvent(ctx, sessionID, assistantEvent); err != nil {
				yield(Event{}, err)
				return
			}
			if assistantEvent.Actions != nil && len(assistantEvent.Actions.StateDelta) > 0 {
				if err := r.session.UpdateState(ctx, sessionID, assistantEvent.Actions.StateDelta); err != nil {
					yield(Event{}, err)
					return
				}
			}
			return
		}

		yield(Event{}, fmt.Errorf("runner exceeded max iterations: %d", r.maxIterations))
	}
}

func buildContents(sess *Session, userContent *Content) []*Content {
	contents := make([]*Content, 0)
	if sess != nil {
		for _, event := range sess.Events {
			if event.Content == nil {
				continue
			}
			contents = append(contents, event.Content)
		}
	}
	if userContent != nil {
		if len(contents) == 0 || contents[len(contents)-1] != userContent {
			contents = append(contents, userContent)
		}
	}
	return contents
}

func (r *Runner) findTool(name string) Tool {
	for _, tool := range r.tools {
		if tool == nil {
			continue
		}
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}

func (r *Runner) executeTools(ctx context.Context, sess *Session, parts []Part) ([]ToolResultPart, error) {
	state := NewSessionState(nil)
	if sess != nil && sess.State != nil {
		state = sess.State
	}

	results := make([]ToolResultPart, 0)
	for _, part := range parts {
		call, ok := part.(ToolCallPart)
		if !ok {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if err := r.callBeforeToolPlugins(ctx, state, &call); err != nil {
			return nil, err
		}

		toolResult := &ToolResult{}
		tool := r.findTool(call.Name)
		if tool == nil {
			toolResult = &ToolResult{
				Content: fmt.Sprintf("tool %q not found", call.Name),
				IsError: true,
			}
		} else {
			execResult, err := tool.Execute(ctx, call.Arguments)
			if err != nil {
				toolResult = &ToolResult{
					Content: err.Error(),
					IsError: true,
				}
			} else if execResult == nil {
				toolResult = &ToolResult{
					Content: "tool returned nil result",
					IsError: true,
				}
			} else {
				toolResult = execResult
			}
		}

		if err := r.callAfterToolPlugins(ctx, state, &call, toolResult); err != nil {
			return nil, err
		}

		results = append(results, ToolResultPart{
			CallID:  call.ID,
			Name:    call.Name,
			Content: toolResult.Content,
			IsError: toolResult.IsError,
		})
	}

	return results, nil
}

func (r *Runner) callBeforeToolPlugins(ctx context.Context, state *SessionState, call *ToolCallPart) error {
	for _, plugin := range r.plugins {
		if plugin == nil {
			continue
		}
		if err := plugin.BeforeTool(ctx, state, call); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) callAfterToolPlugins(ctx context.Context, state *SessionState, call *ToolCallPart, result *ToolResult) error {
	for _, plugin := range r.plugins {
		if plugin == nil {
			continue
		}
		if err := plugin.AfterTool(ctx, state, call, result); err != nil {
			return err
		}
	}
	return nil
}
