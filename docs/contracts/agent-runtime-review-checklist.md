# Agent Runtime Review Checklist

## Boundaries

- [ ] Runtime does not own Conversation, Planner, Registry, Gateway, or AG-UI.
- [ ] official A2A/MCP adapters are used or migration adapter is explicit.
- [ ] business prompt/handler is outside generic runtime package.
- [ ] Context is provided by Orchestrator and authorization-bounded.

## Execution

- [ ] cancellation/deadline reaches model and Tools.
- [ ] streaming order and terminal behavior are tested.
- [ ] retry cannot duplicate visible output.
- [ ] Tool schema, permission, timeout, and output limits exist.
- [ ] Artifact output follows active Contract.

## Operations

- [ ] deterministic Mock mode.
- [ ] health/readiness distinction.
- [ ] configuration validation.
- [ ] secrets and provider payload redacted.
- [ ] invocation/Agent/model/Tool telemetry.
- [ ] concurrency/race coverage.
