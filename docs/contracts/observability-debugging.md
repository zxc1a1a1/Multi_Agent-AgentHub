# Observability and Debugging Contract

**Status:** Active
**Version:** AgentHub 2.0
**Preferred foundation:** OpenTelemetry

## 1. Purpose

Observability must make an AgentHub request diagnosable across:

```text
Frontend
Gateway
Orchestrator
Planner
Context Manager
Agent Registry
A2A Agent
model
Tool/MCP
persistence
Artifact
Preview
```

without exposing secrets, private reasoning, or unrestricted user content.

## 2. Correlation model

Core identifiers:

```text
requestId
traceId
conversationId
messageId
runId
planId
planVersion
stepId
invocationId
agentId
agentVersion
skillId
contextSnapshotId
artifactId
artifactVersion
toolCallId
providerRequestId
errorCode
```

IDs are propagated where relevant, not indiscriminately added to metric labels.

## 3. Trace boundaries

Recommended spans:

```text
gateway.request
gateway.authorize
orchestrator.create_plan
planner.generate
plan.validate
plan.repair
plan.wait_for_confirmation
orchestrator.execute
context.build
context.retrieve
registry.catalog
registry.health_check
a2a.invoke
a2a.stream
model.request
tool.call
artifact.commit_version
preview.load
persistence.transaction
event.replay
```

Waiting for user confirmation may be represented as a Run state/event rather than a continuously open infrastructure span.

## 4. Structured logs

Minimum fields when relevant:

```text
timestamp
level
service
environment
message
requestId
traceId
conversationId
runId
planId/version
stepId
invocationId
agentId/version
artifactId/version
eventSequence
errorCode
retryable
durationMs
```

Rules:

- stable machine-readable keys;
- no raw Authorization header;
- no Agent credential;
- no full provider body;
- no private system prompt;
- no chain-of-thought;
- no attachment/full Artifact content by default;
- safe hashes/IDs instead of content where sufficient.

## 5. Metrics

Metrics focus on bounded labels.

### Product/run

```text
agenthub_runs_total{mode,status}
agenthub_run_duration_seconds{mode,status}
agenthub_plan_generation_seconds{result}
agenthub_plan_validation_total{result,error_code}
agenthub_plan_confirmation_seconds{outcome}
agenthub_agent_selection_total{mode,result}
```

### Context

```text
agenthub_context_estimated_tokens{use_case}
agenthub_context_compaction_total{result}
agenthub_context_retrieval_seconds{scope,result}
agenthub_context_sources_count{source_type}
```

Do not label metrics with Conversation/Run/User ID.

### Agent/A2A/Tool

```text
agenthub_agent_invocations_total{agent_source,status}
agenthub_agent_invocation_duration_seconds{agent_source,status}
agenthub_a2a_stream_interruptions_total{agent_source}
agenthub_tool_calls_total{tool_class,status}
agenthub_tool_duration_seconds{tool_class,status}
```

Dynamic Agent IDs may be high-cardinality; use source/trust class in global metrics and exact ID in traces/logs.

### Artifact/preview

```text
agenthub_artifact_versions_total{artifact_type,editor_type}
agenthub_artifact_conflicts_total{artifact_type}
agenthub_preview_load_seconds{template,result}
agenthub_preview_runtime_errors_total{template,error_class}
```

### Provider/cost

```text
agenthub_model_requests_total{use_case,provider,model,status}
agenthub_model_input_tokens_total{use_case,provider,model}
agenthub_model_output_tokens_total{use_case,provider,model}
agenthub_model_request_duration_seconds{use_case,provider,model,status}
```

Model label must be bounded by configured registry.

## 6. Event replay diagnostics

Track:

- highest persisted sequence;
- client `Last-Event-ID`;
- replay count;
- live attachment time;
- duplicate reducer drops;
- missing/invalid sequence;
- terminal event.

Event payload content is not copied into metrics.

## 7. Error taxonomy

Failures must be distinguishable:

```text
authorization
planning
validation
confirmation
context
registry
a2a
model
tool
persistence
artifact
preview
cancellation
configuration
```

Public error, internal error, trace status, and Run outcome map consistently.

## 8. Debug bundle

A user/operator may explicitly generate a bounded support bundle containing:

- service/version/config names without secrets;
- Run/Plan state;
- sanitized event metadata;
- Agent/Artifact/Context IDs and hashes;
- selected structured logs;
- health status;
- test/diagnostic results.

It excludes:

- credentials;
- bearer tokens;
- private prompts;
- chain-of-thought;
- full attachments/Artifacts unless separately authorized;
- unrestricted database dump.

Bundle generation is audited and retention-limited.

## 9. Sampling and retention

- errors and security denials may use elevated sampling.
- successful high-volume streams use bounded sampling.
- metric retention differs from content/event retention.
- production defaults avoid full body capture.
- user deletion/purge policy applies to linked content telemetry where required.

## 10. Required tests

- trace propagation Gateway→Orchestrator→A2A;
- Plan/Context/Agent/Artifact correlation;
- high-cardinality label review;
- secret and content redaction;
- partial failure diagnosis;
- event replay diagnosis;
- model token/cost accounting;
- preview error metrics;
- debug bundle allowlist;
- deletion/retention behavior.
