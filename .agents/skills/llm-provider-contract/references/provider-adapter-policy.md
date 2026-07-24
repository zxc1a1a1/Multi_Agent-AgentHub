# Provider Adapter Policy

A Provider Adapter normalizes:

```text
request
stream lifecycle
final response
structured output
usage
provider request ID
retryability
safe errors
```

Provider SDK types stay inside the adapter package.

The adapter does not own:

```text
Conversation
Context selection
Plan validation
Agent Registry
Artifact persistence
AG-UI event mapping
```

Provider-specific state may be stored as optional optimization metadata. Local AgentHub state remains authoritative.
