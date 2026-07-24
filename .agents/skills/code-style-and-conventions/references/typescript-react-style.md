# TypeScript and React Style

- Enable strict TypeScript.
- Treat external JSON as `unknown` and validate it.
- Keep API/SSE clients outside generic presentation components.
- Reducers are deterministic and idempotent by Event ID.
- Hooks have explicit cancellation and cleanup.
- Avoid broad `any`, unsafe casts, and hidden global mutable state.
- Conversation, Run, PlanVersion, AgentVersion, and ArtifactVersion remain distinct types.
- Generated project code never executes in the main application origin.
- Accessibility and keyboard behavior are part of component completion.
