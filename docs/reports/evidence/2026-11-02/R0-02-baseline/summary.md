# R0-BASELINE R0-02 integration evidence

- task: R0-02
- testId: R0-BASELINE
- commit: 7193a89f493be9c46d3d52083d738c7ece1d4b1d (dev)
- go: go version go1.26.5 windows/amd64
- started: 2026-11-02T08:56:49Z
- ended: 2026-11-02T08:57:31Z
- duration: 42s
- packages: ./services/orchestrator/... ./services/gateway/...
- run pattern: ^TestIntegration
- go test exit: 1

## results

| metric | count |
|---|---|
| total tests | 33 |
| PASS | 27 |
| FAIL | 6 |
| SKIP | 0 |
| incomplete | 0 |
| interaction rows | 0 |
| overall | FAIL |

## per test

| test | status | elapsed(s) |
|---|---|---|
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationMemoryAdminAPI | ok | 3.840 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory#TestIntegrationRedisRecallCacheUsesEpoch | ok | 1.660 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationDuplicateRequestIsIdempotent | ok | 0.270 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationConcurrentSeqAndStaleFencing | ok | 0.850 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRunInjectsAuthorizedMemory | ok | 1.010 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationCheckpointFencing | ok | 0.320 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationApprovalConsumedOnce | ok | 0.720 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRunLedgerLifecycleMySQL | ok | 1.350 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationToolResultUnknown | ok | 0.560 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationFailedStateChangeLeavesNoHalfResult | ok | 0.510 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationPlanApprovalLifecycle | ok | 0.620 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationOutboxRelayMySQLRedis | error | 0.030 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationLeaseRenewalAndStaleFencing | error | 0.440 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationMemoryLifecycle | ok | 1.520 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRunSignals | error | 0.120 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationLeaseReleaseAllowsImmediateTakeover | ok | 0.350 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRunTaskRefsAndCancel | ok | 0.300 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationHITLConfirmIdempotency | error | 0.110 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRouteDecisions | ok | 0.110 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRouteMetrics | error | 0.180 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationTwoOrchestratorsHITLMySQL | error | 1.080 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationMemoryScopeIsolation | ok | 1.060 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/migration#TestIntegrationSQLiteToMySQLMigration | ok | 1.220 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/projection#TestIntegrationWorkerRecoversAgentTurns | ok | 1.680 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/projection#TestIntegrationTwoWorkersDoNotDuplicateTurns | ok | 1.340 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationSQLiteToMySQLFieldMapping | ok | 0.310 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLProjectionOffsetIsIdempotent | ok | 0.660 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLStoreCRUD | ok | 0.650 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationRecordChatRequestIdempotent | ok | 0.380 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLStoreSurvivesReopen | ok | 0.580 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationConversationIsolation | ok | 0.610 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationPublicEventJournal | ok | 0.510 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLConversationParticipants | ok | 0.790 |

## skipped tests (must be explained)

- none

## failed tests

  {"schemaVersion":2,"evidenceId":"ev_20261102_R0_BASELINE_012","testId":"R0-BASELINE","planTaskId":"R0-02","layer":"middleware","actor":"tool","action":"test_case","target":"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationOutboxRelayMySQLRedis","status":"error","runId":"","conversationId":"","traceId":"","taskId":"","messageId":"","errorCode":"","elapsedSec":0.030,"detail":{"goStatus":"fail"},"recordedAt":"2026-11-02T08:57:31Z"}
  {"schemaVersion":2,"evidenceId":"ev_20261102_R0_BASELINE_013","testId":"R0-BASELINE","planTaskId":"R0-02","layer":"middleware","actor":"tool","action":"test_case","target":"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationLeaseRenewalAndStaleFencing","status":"error","runId":"","conversationId":"","traceId":"","taskId":"","messageId":"","errorCode":"","elapsedSec":0.440,"detail":{"goStatus":"fail"},"recordedAt":"2026-11-02T08:57:31Z"}
  {"schemaVersion":2,"evidenceId":"ev_20261102_R0_BASELINE_015","testId":"R0-BASELINE","planTaskId":"R0-02","layer":"middleware","actor":"tool","action":"test_case","target":"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRunSignals","status":"error","runId":"","conversationId":"","traceId":"","taskId":"","messageId":"","errorCode":"","elapsedSec":0.120,"detail":{"goStatus":"fail"},"recordedAt":"2026-11-02T08:57:31Z"}
  {"schemaVersion":2,"evidenceId":"ev_20261102_R0_BASELINE_018","testId":"R0-BASELINE","planTaskId":"R0-02","layer":"middleware","actor":"tool","action":"test_case","target":"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationHITLConfirmIdempotency","status":"error","runId":"","conversationId":"","traceId":"","taskId":"","messageId":"","errorCode":"","elapsedSec":0.110,"detail":{"goStatus":"fail"},"recordedAt":"2026-11-02T08:57:31Z"}
  {"schemaVersion":2,"evidenceId":"ev_20261102_R0_BASELINE_020","testId":"R0-BASELINE","planTaskId":"R0-02","layer":"middleware","actor":"tool","action":"test_case","target":"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRouteMetrics","status":"error","runId":"","conversationId":"","traceId":"","taskId":"","messageId":"","errorCode":"","elapsedSec":0.180,"detail":{"goStatus":"fail"},"recordedAt":"2026-11-02T08:57:31Z"}
  {"schemaVersion":2,"evidenceId":"ev_20261102_R0_BASELINE_021","testId":"R0-BASELINE","planTaskId":"R0-02","layer":"middleware","actor":"tool","action":"test_case","target":"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationTwoOrchestratorsHITLMySQL","status":"error","runId":"","conversationId":"","traceId":"","taskId":"","messageId":"","errorCode":"","elapsedSec":1.080,"detail":{"goStatus":"fail"},"recordedAt":"2026-11-02T08:57:31Z"}

## interaction coverage (plan §5.5)

| interaction | record file | rows | coverage |
|---|---|---|---|
| test case result | events.jsonl | 33 | yes |
| middleware interactions | interactions.jsonl | 0 | not_applicable: no test in this selection emitted interaction rows |
| user actions | - | 0 | not_applicable: gate script runs no browser code |
| SSE / AG-UI events | - | 0 | not_applicable: no SSE client in this selection |

## redaction

- DSN and Redis address values are replaced with `<redacted-dsn>` / `<redacted-addr>` in all logs.
- Environment values are read only through `ENVIRON` inside awk, so they never reach argv or evidence files.
