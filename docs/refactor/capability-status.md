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
| Dynamic Agent registration | UNKNOWN | pending | implement fully |
| Dynamic Agent persistence | UNKNOWN | pending | required |
| Agent update/delete/enable/disable | UNKNOWN | pending | required |
| Agent refresh/check | UNKNOWN | pending | required |
| Gateway Agent APIs | UNKNOWN | pending | proxy only |
| Frontend Agent management | UNKNOWN | pending | required |
| Dispatcher uses a2a.Client | UNKNOWN | pending | required |
| A2A send | UNKNOWN | pending | P1 |
| A2A get task | UNKNOWN | pending | P1 |
| A2A cancel task | UNKNOWN | pending | P1 |
| ToolResult to A2A | UNKNOWN | pending | P1 |
| PostgreSQL/Redis | DEFERRED | intentionally deferred | do not implement |
| Custom Agent Builder | DEFERRED | intentionally deferred | do not implement |
