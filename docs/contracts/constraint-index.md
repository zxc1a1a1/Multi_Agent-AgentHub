# Constraint Index

Only ACTIVE constraints are binding.
CANDIDATE constraints are reference-only.
SUPERSEDED constraints must not be implemented.

| ID | Topic | Status | Rule |
|---|---|---|---|
| ARCH-001 | Gateway boundary | ACTIVE | Gateway must not call LLMs or child agents |
| ARCH-002 | Orchestrator responsibility | ACTIVE | Orchestrator owns registration/planning/execution/dispatch |
| A2A-001 | A2A ownership | ACTIVE | pkg/adk/a2a is the A2A source of truth |
| REG-001 | Dynamic Agent registration | ACTIVE | Full dynamic registration must be implemented early |
| DB-001 | Database scope | ACTIVE | PostgreSQL/Redis/GORM are deferred |
| OLD-001 | Gateway LLM layer | SUPERSEDED | Do not restore |
| OLD-002 | IntentOrchestrator monolith | SUPERSEDED | Do not restore |
| OLD-003 | RulePlanner as primary router | SUPERSEDED | Do not restore |
| OLD-004 | snake_case PDR plan schema | SUPERSEDED | Do not restore |
