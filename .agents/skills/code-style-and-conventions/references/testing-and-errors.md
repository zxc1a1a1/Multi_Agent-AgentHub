# Testing and Error Style

Errors crossing a boundary contain:

```text
stable code
safe message
retryable classification where relevant
correlation ID
protected internal cause only in redacted logs
```

Tests cover:

```text
success
validation failure
authorization denial
timeout/cancellation
idempotency/conflict
partial/terminal stream
restart/replay where relevant
```

Do not snapshot credentials, full prompts, private content, or unstable provider payloads.
