# AgentHub Code Style and Engineering Conventions

**Status:** Active
**Version:** AgentHub 2.0
**Product source:** `docs/pdr/AgentHub-2.0-PDR.md`

## 1. Purpose

This Contract defines language boundaries, code conventions, validation, errors, tests, and AI-generated change requirements.

It does not redefine domain APIs or protocol fields.

## 2. Language boundary

```text
Go
  core online services
  Gateway and Orchestrator
  built-in/managed Agent services
  Agent Runtime
  A2A/MCP/provider/storage/telemetry adapters

TypeScript
  React frontend
  API and AG-UI client
  state reducers
  Artifact workspace
  Sandpack integration

Python
  evaluation
  dataset preparation
  offline analysis
  bounded development/migration/support scripts

SQL
  goose migrations
  sqlc queries
  FTS5 setup
```

A new core online service in Python or another language requires an approved ADR.

Python is not considered weak; it is simply not the default language for the online AgentHub service boundary.

## 3. General rules

- Prefer readable domain code over speculative abstraction.
- Make one coherent change at a time.
- Validate all external data at the trust boundary.
- Reuse active Contract and official SDK types.
- Keep I/O cancellable and time-bounded.
- Keep transactions short and free of external calls.
- Return stable safe errors.
- Log stable identifiers and bounded metadata.
- Test negative and concurrency behavior.
- Do not claim an unexecuted validation passed.

## 4. Go

- run `gofmt`;
- package names are short, lowercase, and cohesive;
- define interfaces at the consumer side;
- use concrete types until a real adapter/test seam exists;
- wrap errors with operation context without leaking secrets;
- pass `context.Context` to I/O and long-running work;
- every external HTTP/model/Tool call has timeout and cancellation;
- goroutines have owner, stop condition, and error path;
- avoid unbounded channels, buffers, retries, and collections;
- avoid database transactions around model/network/Tool calls;
- use `go test -race` for shared Registry, Run, Event, stream, and Artifact state;
- do not duplicate A2A/MCP/OpenTelemetry/active domain types.

## 5. TypeScript

- strict mode;
- external JSON begins as `unknown`;
- validate API, SSE, Agent, Artifact, and local storage payloads;
- avoid broad `any` and unsafe assertions;
- keep network clients separate from generic UI;
- reducers are deterministic and idempotent;
- stream hooks support abort and cleanup;
- represent Conversation, Run, PlanVersion, AgentVersion, ContextSnapshot, and ArtifactVersion distinctly;
- no Provider or Agent credential in browser code;
- generated preview code never executes in the main application origin.

## 6. React

- components use PascalCase;
- Hooks use `useXxx`;
- Props and state are explicit;
- generic UI components do not fetch domain data directly;
- Conversation switching resets or namespaces transient state;
- effects clean up streams/listeners;
- loading, empty, error, canceled, awaiting-confirmation, and conflict states are visible;
- accessibility and keyboard behavior are part of completion;
- preview and unsafe rendering stay in isolated components.

## 7. Python

- use type hints for public functions;
- use `pathlib`;
- provide CLI help and validate arguments;
- use deterministic seeds when relevant;
- dependencies live in an approved `pyproject.toml` or requirements file;
- no embedded credential;
- no uncontrolled network in default tests;
- use temporary directories/fixtures;
- do not mutate repository files without explicit output flags;
- do not introduce a core online service without ADR approval.

## 8. SQL and persistence

- migrations are forward, ordered, and tested from an empty database;
- destructive changes define compatibility/backup;
- query ownership follows the persistence Contract;
- SQL does not spread through handlers;
- use parameters, not string concatenation;
- pagination and indexes are explicit;
- SQLite busy/transaction behavior is tested;
- FTS query input is bounded and escaped through the approved query layer.

## 9. Naming

Use stable domain names:

```text
Conversation
Message
Run
Plan
PlanVersion
PlanStep
AgentInvocation
RegisteredAgent
AgentVersion
ContextSnapshot
Artifact
ArtifactVersion
ArtifactPatch
```

Do not use:

```text
threadId as a substitute for Conversation without a compatibility adapter
Agent for Planner/Synthesizer/Preview
ToolCall for Plan confirmation or Artifact preview
Skill as a substitute for Agent
```

Identifiers use the field style of the owning contract/API and remain consistent across boundaries.

## 10. Errors and logs

Boundary error:

```text
code
safe message
retryable when relevant
correlation ID
protected internal cause in redacted logs
```

Logs exclude:

- credentials and Authorization headers;
- private prompts and private reasoning;
- full Message/Attachment/Artifact content by default;
- unsafe internal URL;
- raw provider/Agent error body.

## 11. Comments and documentation

Comments explain:

- invariant;
- ownership;
- security assumption;
- non-obvious retry/idempotency behavior;
- compatibility decision.

Do not narrate obvious syntax.

Markdown:

- continuous heading hierarchy;
- valid code blocks;
- precise active/legacy terminology;
- no unsupported test claim.

JSON/YAML:

- parseable;
- no secret;
- Schema uses Draft 2020-12 unless an active Contract says otherwise.

## 12. Generated and AI-assisted changes

Generated code must:

- stay inside approved scope;
- not introduce unrelated dependencies or rewrites;
- compile/typecheck where applicable;
- include real error handling;
- contain no placeholder production path;
- be formatted;
- include tests or a stated blocker;
- report exact validation.

Do not include caches, model files, dependency directories, local databases, or generated binaries unless they are the explicit requested artifact.

## 13. Completion evidence

Report:

```text
changed files
reason
formatter/typecheck/lint
tests
security negatives
migration/compatibility
commands not run and reason
known gaps
```
