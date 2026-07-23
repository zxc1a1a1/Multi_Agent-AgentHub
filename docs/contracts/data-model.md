# AgentHub 2.0 Data Persistence Contract

**Status:** Active  
**Profile:** `v2-single-node-sqlite`  
**Supersedes:** the v1.0 MySQL-first data model as the active default  

## 1. Purpose

This Contract defines the persistence model for:

- multi-conversation IM;
- message and Run history;
- confirmed LLM planning;
- Agent invocation and event replay;
- context summaries and snapshots;
- registered Agent lifecycle;
- artifact versioning;
- full-text conversation search.

## 2. Default storage profile

```text
database       SQLite
journal_mode   WAL
data access    sqlc
migrations     goose
full text      SQLite FTS5
deployment     single-node / Docker Compose
```

A future PostgreSQL profile may implement the same repository interfaces. A transport or database migration requires an explicit Contract/ADR; it is not implied by old redesign documents.

## 3. General conventions

- Table names: plural `snake_case`.
- Column names: `snake_case`.
- API fields: `camelCase`.
- IDs: opaque string/UUID/ULID consistently selected by implementation.
- Timestamps: UTC.
- User-visible mutable resources include `created_at`, `updated_at`, and where applicable `deleted_at`.
- Immutable versions include `created_at` and content/hash metadata.
- JSON columns MUST have a documented schema.
- Foreign keys MUST be enabled in SQLite.
- Query results MUST have deterministic ordering.

## 4. Core tables

### 4.1 Identity and conversation

```text
users
conversations
conversation_members
messages
attachments
```

### 4.2 Run and planning

```text
runs
plans
plan_versions
agent_invocations
events
run_agent_snapshots
```

### 4.3 Artifact and context

```text
artifacts
artifact_versions
conversation_summaries
context_snapshots
context_chunks
```

### 4.4 Agent Registry

```text
registered_agents
agent_versions
agent_health_checks
agent_credentials
```

## 5. Entity contracts

## 5.1 users

Minimum fields:

```text
id
display_name
preferences_json
created_at
updated_at
deleted_at
```

Even a single-user profile retains `user_id` foreign keys.

## 5.2 conversations

```text
id
user_id
title
mode
response_mode
status
context_policy_json
version
last_message_at
pinned_at
archived_at
deleted_at
created_at
updated_at
```

`mode`:

```text
direct
manual_multi
auto
```

`status`:

```text
active
archived
deleted
```

`version` is used for optimistic concurrency.

## 5.3 conversation_members

```text
conversation_id
member_type
member_id
role
joined_at
left_at
```

Planner, Synthesizer, and Context Manager MUST NOT be stored as Agent members.

A removed Agent may remain referenced by historical membership rows and snapshots.

## 5.4 messages

```text
id
client_message_id
conversation_id
run_id
sender_type
sender_id
message_type
content_text
content_json
reply_to_message_id
status
sequence
created_at
updated_at
deleted_at
```

Constraints:

- `(conversation_id, sequence)` unique.
- `(conversation_id, client_message_id)` unique when client ID is present.
- streaming deltas live in `events`; the committed final user-visible content lives in `messages`.

## 5.5 runs

```text
id
conversation_id
trigger_message_id
mode
status
plan_id
confirmed_plan_version
context_snapshot_id
started_at
finished_at
error_code
metadata_json
created_at
updated_at
```

Run status:

```text
pending
planning
awaiting_confirmation
executing
synthesizing
completed
partial_failure
failed
canceled
```

The default policy permits one active Run per Conversation.

## 5.6 plans

```text
id
run_id
current_version
status
created_at
updated_at
```

## 5.7 plan_versions

```text
plan_id
version
goal
mode
selected_agents_json
steps_json
constraints_json
response_mode
generated_by
planner_metadata_json
validation_result_json
created_at
confirmed_at
confirmed_by
content_hash
```

Constraints:

- `(plan_id, version)` unique.
- confirmed content is immutable.
- execution references exact `plan_id + version`.

## 5.8 agent_invocations

```text
id
run_id
plan_step_id
agent_id
agent_version
skill_id
status
input_context_snapshot_id
output_message_id
started_at
finished_at
token_usage_json
provider_request_id
error_code
error_json
```

## 5.9 events

```text
id
conversation_id
run_id
invocation_id
event_type
payload_json
sequence
created_at
```

Constraints:

- `(run_id, sequence)` unique.
- sequence increases monotonically within a Run.
- Event payloads are replayable and versioned.

## 5.10 attachments

```text
id
conversation_id
message_id
filename
mime_type
size_bytes
storage_uri
content_hash
parse_status
extracted_text_uri
created_at
deleted_at
```

Binary data SHOULD remain outside ordinary SQLite rows. Metadata and references are stored in the database.

## 5.11 artifacts

```text
id
conversation_id
run_id
source_invocation_id
type
name
current_version
status
created_at
updated_at
deleted_at
```

## 5.12 artifact_versions

```text
artifact_id
version
content_uri
content_json
base_version
change_summary
created_by_type
created_by_id
content_hash
created_at
```

Constraints:

- `(artifact_id, version)` unique.
- version rows are immutable.
- patch application validates `base_version`.

## 5.13 conversation_summaries

```text
id
conversation_id
version
covered_from_sequence
covered_to_sequence
summary_text
open_tasks_json
key_decisions_json
active_artifact_refs_json
model_metadata_json
created_at
```

Summary generation MUST NOT delete or rewrite original Messages.

## 5.14 context_snapshots

```text
id
conversation_id
run_id
invocation_id
agent_id
agent_version
policy_version
system_context_hash
message_ids_json
summary_ids_json
artifact_refs_json
retrieved_chunk_ids_json
estimated_tokens
created_at
```

This records selected context provenance. It MUST NOT store private model chain-of-thought.

## 5.15 context_chunks

```text
id
conversation_id
source_type
source_id
chunk_index
content_text
content_hash
search_text
created_at
deleted_at
```

Vector fields are optional future extensions. FTS5 is the default 2.0 retrieval profile.

## 5.16 registered_agents

```text
id
name
description
owner_id
registration_source
status
visibility
trust_level
current_version
created_at
updated_at
removed_at
```

`registration_source`:

```text
builtin
config
remote
managed_future
```

`status`:

```text
pending
validating
active
unhealthy
disabled
rejected
removed
```

## 5.17 agent_versions

```text
agent_id
version
card_json
card_hash
endpoint
protocol
capabilities_json
input_modes_json
output_modes_json
streaming
auth_type
created_at
activated_at
deprecated_at
```

An AgentCard refresh creates or records a new version/hash; historical Runs retain their snapshot.

## 5.18 agent_health_checks

```text
id
agent_id
agent_version
status
latency_ms
error_code
error_message
checked_at
```

Health is observational state and MUST remain separate from lifecycle status.

## 5.19 agent_credentials

```text
id
agent_id
credential_type
encrypted_payload
key_version
created_at
updated_at
rotated_at
```

Credential content MUST NOT be returned by ordinary read APIs or included in Planner Catalog.

## 5.20 run_agent_snapshots

```text
run_id
agent_id
agent_version
card_hash
endpoint_snapshot
capabilities_json
created_at
```

A Run snapshot explains historical planning and invocation after Registry changes.

## 6. Ownership and write boundaries

Domain ownership:

```text
Gateway domain:
  users, conversations, members, messages, attachments, public object authorization

Orchestrator domain:
  runs, plans, plan_versions, invocations, events, context snapshots

Registry domain:
  registered_agents, agent_versions, health, credentials, run snapshots

Artifact domain:
  artifacts, artifact_versions
```

In the SQLite single-node profile, domains may share a database file through a common storage package, but MUST:

- use explicit repository interfaces;
- use short transactions;
- avoid cross-domain ad hoc SQL;
- enable WAL;
- configure `busy_timeout`;
- apply bounded retry for `SQLITE_BUSY`;
- appoint one migration owner/process;
- avoid long model/tool calls inside transactions.

## 7. Transaction boundaries

Examples:

### Send user message

One transaction:

- validate Conversation version;
- insert user Message;
- update Conversation `last_message_at`;
- create Run;
- commit.

### Confirm PlanVersion

One transaction:

- validate current version;
- verify awaiting-confirmation state;
- record confirmation;
- set Run status;
- commit.

### Commit Agent response

One transaction:

- insert final Message;
- update invocation status;
- insert artifact metadata/version if any;
- update Run state when complete;
- update Conversation activity;
- commit.

Streaming Event insertion may use small independent batches.

## 8. Idempotency and concurrency

- Message submission uses `client_message_id`.
- Plan confirmation is idempotent for the same version and user.
- Artifact patches require `base_version`.
- Conversation update uses optimistic `version`.
- One active Run per Conversation is the default 2.0 policy.
- Event replay uses `(run_id, sequence)`.

## 9. Indexes

Minimum indexes:

```text
conversations(user_id, last_message_at DESC)
conversations(user_id, archived_at, deleted_at)
messages(conversation_id, sequence)
messages(conversation_id, client_message_id)
runs(conversation_id, created_at DESC)
events(run_id, sequence)
agent_invocations(run_id, started_at)
artifacts(conversation_id, updated_at DESC)
registered_agents(status, visibility, updated_at)
agent_versions(agent_id, version)
agent_health_checks(agent_id, checked_at DESC)
context_chunks(conversation_id, source_type, source_id)
```

## 10. Full-text search

FTS5 indexes:

- Conversation title;
- user and final Agent Message text;
- Conversation Summary;
- Artifact name and searchable text metadata.

Search MUST apply user and Conversation authorization before returning results.

## 11. Deletion and retention

### Archive

Keeps all data and searchability according to UI filters.

### Soft delete

Hides resources from default queries but preserves recoverability and audit references.

### Purge

Must explicitly remove:

- attachment binaries;
- artifact contents;
- FTS rows;
- credentials where applicable;
- dependent records according to retention policy.

Historical Run integrity and legal retention requirements MUST be considered before purge.

## 12. Provider state

Optional fields such as:

```text
provider_conversation_id
previous_response_id
provider_cache_key
```

are optimization metadata only. AgentHub MUST be able to reconstruct context from local data after provider change.

## 13. Security

- object-level authorization on all reads/writes;
- encrypted Agent credentials;
- no secrets in event payloads, Plan metadata, context snapshot, or logs;
- content hashes for attachment/artifact integrity;
- sanitized database errors;
- cross-conversation queries require explicit authorization.

## 14. Required tests

- migrate empty database to current version;
- concurrent message idempotency;
- one active Run per Conversation;
- PlanVersion immutability;
- stale Plan confirmation rejection;
- Artifact base version conflict;
- Event sequence replay;
- Registry version and snapshot retention;
- FTS authorization filtering;
- archive, soft delete, and purge behavior;
- service restart persistence;
- `SQLITE_BUSY` bounded retry behavior.
