# LLM Use-Case Policy

Stable AgentHub 2.0 use cases:

```text
planner
code_agent
web_agent
synthesizer
conversation_summary
auto_title
```

Each policy declares:

```text
provider/model
temperature
max output tokens
timeout
retry
structured output requirement
fallback candidates
prompt template version
usage tags
```

Rules:

- Planner requires structured output and local Plan validation.
- Summary and Auto Title use bounded context and cannot mutate Conversation facts silently.
- Agent use cases are handler policy; they do not redefine AgentCard.
- Synthesizer receives bounded Agent outputs and does not become an Agent.
