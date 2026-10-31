# Conversation Participants Contract

**Status:** Active (v3 P0 frozen, A0-02)
**Owner:** Gateway store + HTTP API
**Last updated:** 2026-10-31
**Implementation:** `services/gateway/store/migrations/003_conversation_participants.sql`, `services/gateway/store/store.go`, `services/gateway/httpapi/server.go`

## 1. Scope

This contract freezes group-chat membership and turn ordering. It covers the
Gateway-owned conversation roster; it does not define Orchestrator execution
state. Plan participants (`plan.PlanParticipant`) remain the planner's proposal
and are authorized separately.

## 2. Truth tables

### 2.1 `conversations`

| Column | Type | Rule |
|---|---|---|
| `id` | `VARCHAR(64)` | Primary key |
| `conversation_type` | `VARCHAR(32)` | `single` or `group`; default `single` |
| `agent_name` | `VARCHAR(128) NULL` | Compatibility display name; always the first roster member when a roster exists |
| `user_id` | `VARCHAR(128)` | Owner; required by the create API |

`conversation_type=group` is written when the user selects more than one agent,
or explicitly sends `"type":"group"`. Changing the roster to one member writes
`single`.

### 2.2 `conversation_participants`

| Column | Type | Rule |
|---|---|---|
| `conversation_id` | `VARCHAR(64)` | Part of the primary key; `ON DELETE CASCADE` |
| `agent_name` | `VARCHAR(128)` | Part of the primary key |
| `role` | `VARCHAR(32)` | `owner` for the first member, otherwise `member` |
| `selected` | `TINYINT(1)` | Default `1`; reserved for per-turn selection |
| `required` | `TINYINT(1)` | Default `0`; required members cannot be deselected |
| `sort_order` | `INT` | Stable display/turn order, 0-based |
| `created_at` / `updated_at` | `DATETIME(3)` | Server clock (UTC) |

Uniqueness: `PRIMARY KEY (conversation_id, agent_name)`. Replaying a create or
update cannot duplicate a member; the roster is replaced atomically
(`DELETE` + ordered `INSERT` in one transaction).

## 3. Permissions

- Only the API caller's `userId` is stored; there is no cross-user sharing.
- The user owns the roster. The platform never appends a member the user did
  not select; planner output cannot mutate the Gateway roster.
- Agent names are trimmed, deduplicated, limited to 128 bytes, and capped at 32
  members per conversation.
- The roster is advisory for routing. The authoritative execution boundary is
  still `AllowedAgents` in the Orchestrator plan.

## 4. API surface

| Method | Path | Behavior |
|---|---|---|
| `POST` | `/api/conversations` | Accepts `userId`, optional `agentName`, `agentNames[]`, `type`; creates conversation and roster |
| `GET` | `/api/conversations/{id}/participants` | Returns ordered roster; 404 for unknown conversation |
| `PATCH` | `/api/conversations/{id}/participants` | Replaces roster; recomputes `conversation_type` |
| `PATCH` | `/api/conversations/{id}` | Same roster replacement (no title fields yet) |

Response `Conversation` includes `type` and `participants[]`.
`PATCH` requires a non-empty `agentNames`.

## 5. Producers and consumers

| Producer | Data | Consumer |
|---|---|---|
| Gateway create/update API | `conversations.conversation_type`, roster | Frontend `ChatWindow`, conversation list |
| Frontend `ChatWindow` | selected agent names | `PATCH .../participants`, local cache `agenthub:selectedAgents:<conversationId>` |
| Gateway replay / history | no change | reads roster on refresh |

The Orchestrator does **not** read this table; it receives `selectedAgentNames`
and `executionPath` on the run request.

## 6. Turn history

Each `AGENT_TURN_*` group is persisted as its own assistant message keyed by the
turn `messageId`; see `docs/contracts/agui-events.md` §AGENT_TURN and
`docs/contracts/event-replay.md` §5. History must not collapse multiple agents
into one message.

## 7. Tests

- `services/gateway/store/store_test.go` — memory roster round-trip, replacement,
  cascade delete, unknown conversation.
- `services/gateway/store/mysql_integration_test.go` —
  `TestIntegrationMySQLConversationParticipants` (requires
  `AGENTHUB_MYSQL_TEST_DSN`; CI runs it in `v3-integration-gates.yml`).
- `services/gateway/httpapi/conversation_participants_test.go` — create group,
  dedupe, list, replace, 400/404.
- `frontend/src/components/ChatWindow.test.tsx` — roster restore from server and
  local-cache fallback.
