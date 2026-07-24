# Go Style

- Use `gofmt`.
- Keep packages small and domain-cohesive.
- Define interfaces at the consumer boundary.
- Prefer concrete types until a test or adapter seam is required.
- Wrap errors with safe operation context; preserve sentinel/code mapping.
- Pass `context.Context` to I/O and long-running work.
- Every external call has timeout/cancellation.
- Goroutines have an owner, stop condition, and error path.
- Avoid holding database transactions across network/model/Tool calls.
- Use race tests for shared Registry, Run, Event, and Artifact state.
- Do not duplicate active Contract or official SDK protocol types.
