package adk

import (
	"context"
	"iter"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
)

// Context provides the runtime API for agent handlers.
//
// Per adk-runtime-contract section 16 (Runtime API):
//   - ctx.StreamText(chunk) → outputs streaming text
//   - ctx.AddArtifact(artifact) → adds a task artifact
//
// Per adk-runtime-contract section 7 (prohibitions):
//   - Handler must NOT directly operate on A2A SSE raw responses
//   - Handler must NOT directly return AG-UI events
//   - Handler must NOT bypass ctx.AddArtifact to produce artifacts
//
// This context hides A2A streaming details from the handler.
type Context struct {
	ctx       context.Context
	execCtx   *a2asrv.ExecutorContext
	artifacts []Artifact
	cancel    context.CancelFunc
	yield     func(a2a.Event, error) bool
}

// NewContext creates a new ADK context for the given execution.
func NewContext(ctx context.Context, execCtx *a2asrv.ExecutorContext, yield func(a2a.Event, error) bool) *Context {
	ctx, cancel := context.WithCancel(ctx)
	return &Context{
		ctx:     ctx,
		execCtx: execCtx,
		cancel:  cancel,
		yield:   yield,
	}
}

// StreamText outputs a text chunk to the stream.
//
// Each call immediately emits an A2A TaskArtifactUpdateEvent so the
// upstream orchestrator can forward it as TEXT_MESSAGE_CONTENT without
// waiting for the handler to complete.
//
// Per streaming-output rules:
//   - Allowed: agent reply text, task progress, code explanation
//   - NOT allowed: large code packages, secrets, stack traces, internal paths
//   - Text stream is NOT the artifact source of truth
func (c *Context) StreamText(chunk string) {
	if c.yield == nil {
		return
	}
	textPart := a2a.NewTextPart(chunk)
	textEvent := a2a.NewArtifactEvent(c.execCtx, textPart)
	textEvent.Artifact.Name = "response"
	textEvent.Artifact.Description = "Agent text response"
	c.yield(textEvent, nil)
}

// AddArtifact adds a task artifact.
//
// Per adk-runtime-contract section 18:
//   - MVP only allows artifact.type = "code"
//   - Artifact schema, storage and preview mapping are handled by artifact-contract
func (c *Context) AddArtifact(artifact Artifact) {
	c.artifacts = append(c.artifacts, artifact)
}

// Context returns the underlying context.Context for HTTP calls, cancellation, etc.
func (c *Context) Context() context.Context {
	return c.ctx
}

// Done returns a channel closed when the context is cancelled.
func (c *Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Cancel cancels the context.
func (c *Context) Cancel() {
	c.cancel()
}

// TaskHandler is the function signature for agent task processing.
// Per task-handler-contract section 2: recommended signature is HandleTask(ctx, task) error.
type TaskHandler func(ctx *Context, messages []a2a.Message) error

// NoopHandler is a TaskHandler that does nothing.
// Useful for testing A2A server endpoints without real LLM or business logic.
func NoopHandler(ctx *Context, messages []a2a.Message) error {
	return nil
}

// ExecuteHandler runs a TaskHandler and converts its outputs to A2A events.
// This bridges the ADK runtime abstraction to the a2a-go AgentExecutor interface.
//
// Event flow per a2a-agent-contract:
//   1. TaskStatusUpdate(Working)
//   2. ArtifactUpdate events (streaming text + code artifacts)
//   3. TaskStatusUpdate(Completed) or TaskStatusUpdate(Failed)
func ExecuteHandler(
	ctx context.Context,
	execCtx *a2asrv.ExecutorContext,
	handler TaskHandler,
) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		// 1. Emit submitted task if new
		if execCtx.StoredTask == nil {
			task := a2a.NewSubmittedTask(execCtx, execCtx.Message)
			if !yield(task, nil) {
				return
			}
		}

		// 2. Emit working status
		workingEvent := a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil)
		if !yield(workingEvent, nil) {
			return
		}

		// 3. Run the handler
		adkCtx := NewContext(ctx, execCtx, yield)
		defer adkCtx.Cancel()

		// Extract messages from the execution context
		var messages []a2a.Message
		if execCtx.Message != nil {
			messages = append(messages, *execCtx.Message)
		}
		if execCtx.StoredTask != nil {
			for _, msg := range execCtx.StoredTask.History {
				if msg != nil {
					messages = append(messages, *msg)
				}
			}
		}

		err := handler(adkCtx, messages)

		if err != nil {
			// Per adk-runtime-contract section 20: errors must be user-safe
			// Do NOT expose stack traces, API keys, internal paths
			safeMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart("An error occurred while processing your request."))
			failedEvent := a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateFailed, safeMsg)
			yield(failedEvent, nil)
			return
		}

		// 4. Emit code artifacts
		for _, art := range adkCtx.artifacts {
			part := art.ToA2APart()
			artEvent := a2a.NewArtifactEvent(execCtx, part)
			artEvent.Artifact.Name = art.Title
			artEvent.Artifact.Description = art.Type + " artifact"
			artEvent.Artifact.Metadata = map[string]any{
				"type":     art.Type,
				"language": art.Metadata["language"],
				"filename": art.Title,
			}
			if !yield(artEvent, nil) {
				return
			}
		}

		// 5. Emit completed status
		completedEvent := a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil)
		yield(completedEvent, nil)
	}
}
