# Provider Retry and Streaming Policy

## Before visible output

Bounded retry or compatible fallback may occur for classified transient failures.

## After visible output

Do not restart the full request by default.

Allowed recovery requires explicit safe semantics such as:

```text
provider-supported resume
deduplicated continuation
new user-visible invocation
```

Streaming lifecycle:

```text
start
zero or more deltas
one terminal end or error
```

Cancellation stops provider work and further emission.
