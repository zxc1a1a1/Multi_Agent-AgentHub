# Capability Status

Status values:
- UNKNOWN
- TRUE_GAP
- FALSE_GAP
- PARTIAL_GAP
- INTENTIONAL
- DEFERRED
- DONE

| Capability | Status | Evidence | Decision |
|---|---|---|---|
| A2A AgentCard source of truth | UNKNOWN | pending | verify first |
| Dynamic Agent registration | DONE | Phase 5 `POST /internal/orchestrator/agents` with `DynamicAgentRegistry.Register` | implemented |
| Dynamic Agent persistence | DONE | Phase 5 `JSONStore` at `ORCHESTRATOR_AGENT_STORE_PATH` | implemented |
| Agent update/delete/enable/disable | DONE | Phase 5 `PATCH/DELETE/POST enable/disable /internal/orchestrator/agents/{name}*` | implemented |
| Agent refresh/check | DONE | Phase 5 `POST refresh/check /internal/orchestrator/agents/{name}/(refresh|check)` | implemented |
| Gateway Agent APIs | DONE | Phase 5 `GET/POST/PATCH/DELETE /api/agents*` proxy to Orchestrator or static fallback | implemented |
| Frontend Agent management | UNKNOWN | pending | required |
| Dispatcher uses a2a.Client | UNKNOWN | pending | required |
| A2A send | UNKNOWN | pending | P1 |
| A2A get task | UNKNOWN | pending | P1 |
| A2A cancel task | UNKNOWN | pending | P1 |
| ToolResult to A2A | UNKNOWN | pending | P1 |
| PostgreSQL/Redis | DEFERRED | intentionally deferred | do not implement |
| Custom Agent Builder | DEFERRED | intentionally deferred | do not implement |
