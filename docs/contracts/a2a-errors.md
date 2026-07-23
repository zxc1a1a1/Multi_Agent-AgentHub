# A2A Error Contract

**Status:** Active
**Version:** AgentHub 2.0

## 1. Error classes

```text
agent_unavailable
agent_disabled
agent_unhealthy
agent_unauthorized
agent_card_fetch_failed
agent_card_invalid
agent_protocol_unsupported
agent_auth_required
task_rejected
task_not_found
task_not_cancelable
task_failed
task_input_required
stream_interrupted
stream_protocol_invalid
artifact_invalid
timeout
canceled
```

## 2. Retry policy

| Error | Retry before visible output | Retry after visible output |
|---|---:|---:|
| connection/temporary unavailable | bounded | no full retry |
| timeout | bounded by policy | no full retry |
| stream interrupted | reconnect/resubscribe if supported | no duplicate restart |
| invalid card/protocol | no | no |
| unauthorized/auth required | no automatic credential guessing | no |
| task rejected/invalid | no | no |
| rate limited | bounded using safe retry guidance | only before visible output |

## 3. Propagation

A2A adapter returns:

```text
code
safeMessage
retryable
remoteTaskId when safe
invocationId
internal cause for protected logs
```

Orchestrator maps to internal events. Gateway maps to public error/AG-UI.

## 4. Redaction

Never expose:

```text
Agent credential
Authorization header
internal endpoint
stack trace
provider response body
DNS/private address details
system prompt
user token
```

## 5. Plan interaction

If a confirmed Agent becomes unavailable:

- fail/pause the Step;
- do not silently replace the Agent;
- propose a Replan when an eligible alternative exists;
- require user confirmation for the new PlanVersion.

## 6. Required tests

- sanitization;
- retry boundary before/after stream;
- cancel/not-cancelable mapping;
- auth-required mapping;
- invalid protocol/card;
- Replan required instead of silent substitution.
