# Dispatcher Resilience Contract

**Status:** Active
**Owner:** AgentHub
**Scope:** `services/orchestrator/dispatcher`

The Orchestrator's A2A dispatcher applies resilience policies when calling
Child Agents. These policies make `timeoutMs` and `fallback` (already present in
`OrchestrationPlan`) operative rather than advisory.

## Per-call timeout

- Each dispatch is bounded by a timeout. Resolution order:
  1. `DispatchInput.TimeoutMs` (from `TaskPlan.TimeoutMs`) when > 0.
  2. Dispatcher default (`WithPerCallTimeout`, env `DISPATCH_TIMEOUT_MS`).
- On timeout the dispatch returns a sanitized error; the executor maps it to
  `ORCHESTRATOR_AGENT_FAILED` (or triggers fallback, see below).

## Retry

- Retries apply only to **idempotent, pre-stream** failures: connection
  failures, request timeouts, and HTTP 5xx.
- Retries do NOT apply to: HTTP 4xx, remote validation errors, or any failure
  that occurs **after the first streamed chunk** has been emitted (avoids
  duplicate user-visible output).
- Backoff is exponential with jitter, capped by `WithRetry(maxAttempts, base)`.
  `maxAttempts` counts the total attempts (1 = no retry). Env: `DISPATCH_MAX_RETRY`.
- Context cancellation aborts retries immediately.

## Circuit breaker

- Per agent URL. After `failThreshold` consecutive failures the breaker opens
  for `cooldown`; calls short-circuit with an unavailable error until cooldown
  elapses, then the breaker half-opens and a success closes it.
- Configured via `WithCircuitBreaker(failThreshold, cooldown)`. Env:
  `DISPATCH_BREAKER_THRESHOLD`, `DISPATCH_BREAKER_COOLDOWN_MS`.

## Fallback (deferred)

> **Status: not yet implemented.** `OrchestrationPlan.Fallback` is currently
> *planner* metadata (e.g. `Reason: "rule_default"` records that planning fell
> back to rules); it is **not** an execution-time degraded-output signal.
> Execution-level fallback (emitting a sanitized degraded message + `degraded`
> run status on task failure) requires a dedicated, non-overloaded signal and
> will be specified in a later phase. Until then, a failed task surfaces as
> `ORCHESTRATOR_AGENT_FAILED` (single) or contributes to `partial_failure`
> (ordered_parallel), unchanged.

## Defaults (deterministic CI)

- All policies are opt-in via dispatcher options. With no options configured
  (the default `NewA2ADispatcher()`), behavior is identical to pre-resilience:
  a single attempt, no breaker, request context timeout only. CI must not depend
  on external services.
