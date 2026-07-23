# Security Boundaries Contract

**Status:** Active
**Version:** AgentHub 2.0

## 1. Purpose

This Contract defines security boundaries for:

- browser and public API;
- Gateway and Orchestrator;
- registered Agents and A2A;
- Agent registration and credentials;
- Planner and confirmation;
- Conversation/context;
- Attachments and Artifacts;
- web preview;
- Tools;
- logs, errors, and audit.

## 2. Trust model

Treat as untrusted:

```text
Frontend
user input
LLM output
remote Agent
AgentCard
web page/search result
Attachment
Artifact
Tool output
generated HTML/JavaScript
```

A built-in Agent is more operationally trusted than an arbitrary remote Agent but still does not bypass authorization, context, Tool, or output validation.

## 3. Public service boundary

- Frontend communicates with Gateway only.
- Orchestrator, A2A endpoints, Registry stores, and provider endpoints are not browser-accessible.
- Gateway authenticates every `/api/**` route except explicitly public health.
- object-level authorization is applied to Conversation and all child resources.
- tokens are not accepted in query strings.
- CORS is allowlisted; wildcard with credentials is forbidden.
- CSRF strategy is explicit for cookie-based profiles.

## 4. Gateway boundary

Gateway:

- authenticates;
- authorizes;
- validates request shape/size;
- handles idempotency;
- sanitizes errors;
- passes a sanitized principal.

Gateway MUST NOT:

- call Agent or LLM directly;
- forward browser bearer tokens to Agents;
- trust Agent IDs from Frontend without Registry authorization;
- expose Agent credentials or internal URLs;
- execute generated content.

## 5. Orchestrator boundary

Orchestrator:

- validates Planner output;
- enforces confirmed PlanVersion;
- filters Agent Catalog;
- builds bounded context;
- applies Agent credentials in the client adapter only;
- converts A2A output to internal safe events.

Orchestrator MUST NOT:

- trust LLM Agent/Skill selection;
- send full internal prompts to remote Agents;
- mix Conversation context;
- silently replace confirmed Agents;
- expose raw remote errors.

## 6. Registered Agent security

### Registration

Remote registration is SSRF-sensitive.

Required controls:

- HTTPS by default in production;
- explicit development exceptions;
- block loopback, link-local, unspecified, multicast, and cloud metadata targets;
- validate all resolved IPs;
- revalidate every redirect target;
- limit redirects;
- mitigate DNS rebinding;
- connect/read/total timeout;
- response byte/depth limit;
- safe JSON/content-type handling;
- audit registration and refresh.

Private-network Agents require an operator-controlled deployment profile.

### Credentials

- credential input is write-only;
- encrypted at rest;
- key version recorded;
- rotated/revoked explicitly;
- never returned through API;
- never included in Planner Catalog, AgentCard snapshot shown to users, Event, ContextSnapshot, or log;
- scoped to Agent endpoint/security scheme.

### Invocation

Registration does not grant:

- local Tool access;
- unrestricted Attachment access;
- cross-conversation history;
- user bearer token;
- provider secret;
- arbitrary network access beyond the registered endpoint.

## 7. Planner and confirmation

LLM output is untrusted.

- Agent/Skill/dependency validation is deterministic.
- Manual Mode cannot add unselected Agents.
- unconfirmed Plan cannot execute.
- confirmation binds exact PlanVersion.
- Replan requires new confirmation.

Plan confirmation approves the displayed orchestration structure. It does not automatically approve high-risk Tool actions.

Separate confirmation/approval is required for policy-defined actions such as:

```text
file overwrite/delete
command execution
deployment
external publish
credential use expansion
sensitive data release
```

## 8. Conversation and context

- all context sources are authorization-checked;
- default retrieval is Conversation-local;
- cross-conversation retrieval requires explicit user/policy enablement;
- ContextSnapshot stores provenance, not secret content or private reasoning;
- summaries cannot convert untrusted Agent/web output into confirmed user facts;
- Pinned Context remains subject to authorization/redaction;
- remote Agents receive least necessary context.

External web/Agent content is data, not instruction. Prompt-injection defenses and source boundaries must be applied.

## 9. Attachment security

- file size limit;
- MIME and extension validation;
- content hash;
- private storage;
- malware/security scan when profile requires;
- no direct execution;
- no automatic full-file prompt injection;
- authorization on upload/read/delete;
- sanitized filename;
- decompression/archive bomb limits.

## 10. Artifact security

- Artifact content is untrusted;
- object-level authorization;
- immutable versions;
- patch `baseVersion` conflict check;
- unknown type safely rendered as text/JSON/download;
- private content uses authorized access;
- metadata cannot contain secrets;
- remote Artifact/File references are fetched only through approved policies.

## 11. Web preview

Generated web code executes only in a sandboxed renderer/iframe.

Required:

- origin/storage isolation from main application;
- no parent token/localStorage access;
- restrictive sandbox/CSP;
- dependency policy;
- network policy;
- no top navigation/popups by default;
- error output sanitized;
- preview lifecycle tied to an authorized ArtifactVersion.

Do not inject generated HTML into the main DOM using unsafe APIs.

## 12. Tool execution

Tools follow least privilege:

- explicit schema;
- Agent/Skill permission;
- timeout;
- output limit;
- audit;
- cancellation;
- workspace/root restrictions where applicable;
- separate high-risk approval.

Remote Agent Skill declaration is not Tool authorization.

## 13. Secrets

Forbidden in source, Contract examples, AgentCard, events, logs, Planner input, context snapshots, and public errors:

```text
API key
bearer token
private key
database credential
service token
OAuth secret
Agent credential
secret header
```

`.env.example` contains names/placeholders only.

## 14. Errors and logs

Public error:

```text
code
safe message
requestId
bounded safe details
```

Protected logs may include correlation and error class, but still redact secrets and sensitive content.

Recommended correlation:

```text
requestId
traceId
conversationId
runId
planId/version
invocationId
agentId/version
artifactId/version
errorCode
```

## 15. Audit

Audit:

- Agent register/refresh/enable/disable/remove;
- credential create/rotate/revoke;
- Plan confirm/reject/Replan;
- high-risk Tool approval;
- Attachment purge;
- Artifact restore;
- cross-conversation retrieval;
- authorization denial where useful.

## 16. Required security tests

- public/internal route isolation;
- object-level authorization;
- token query rejection;
- error/log redaction;
- AgentCard secret rejection/sanitization;
- loopback/link-local/metadata/redirect/DNS-rebinding SSRF cases;
- credential write-only and encryption path;
- Planner Registry bypass rejection;
- unconfirmed execution rejection;
- cross-conversation contamination rejection;
- Attachment size/type/archive checks;
- Artifact version conflict;
- sandbox/CSP/no parent storage access;
- high-risk Tool requires separate approval;
- remote Agent cannot gain Tool permission through registration.
