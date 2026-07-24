# LLM Provider Security

**Status:** Active
**Version:** AgentHub 2.0

## Secret rules

Provider credentials may come from:

```text
environment reference
container secret
secret manager
approved local development secret store
```

Configuration stores the reference name, not the value.

Never write a real credential to:

```text
Git
AgentCard
Plan/PlanVersion
Message
Event
Artifact
ContextSnapshot
ordinary database field
frontend environment
log/trace/metric
debug bundle
test fixture
```

User and browser bearer tokens are not Provider credentials.

## Content and reasoning

Ordinary logs and telemetry exclude:

- full system/developer prompt;
- unredacted user Message;
- Attachment/Artifact body;
- provider raw request/response;
- private model reasoning or chain-of-thought.

AgentHub requests final answers, bounded explanations, and structured outputs.

Usage-only fields such as reasoning token counts may be recorded when they contain no reasoning content.

## Trust boundary

```text
Gateway
  X Provider SDK / Provider credential

authorized backend module
  -> Provider Router
  -> Provider Adapter
  -> external Provider
```

Provider output remains untrusted until parsed, validated, and mapped into the owning domain.

## Base URL and proxy

Custom/provider-compatible base URLs require:

- explicit configuration;
- TLS policy;
- no browser exposure;
- credential scoping;
- timeout and response-size limits;
- safe redirect/network policy where applicable.

A Provider proxy cannot silently receive credentials intended for another trust boundary.

## Errors

Public/provider-normalized errors expose:

```text
safe code
safe message
retryable
correlation ID
```

They do not expose:

```text
credential
Authorization header
internal URL with secret query
provider raw body
private prompt
private reasoning
```

## Required tests

- credential reference not value;
- redacted configuration/log/error;
- Gateway has no Provider secret path;
- custom base URL policy;
- raw provider error sanitization;
- prompt/content/reasoning absence from logs;
- debug bundle allowlist.
