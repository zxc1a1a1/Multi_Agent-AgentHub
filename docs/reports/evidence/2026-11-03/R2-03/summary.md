# R2-03 R2-03 integration evidence

- task: R2-03
- testId: R2-03
- commit: 29cf7779295608081e90dc5daaabba860a45807a (dev)
- go: go version go1.26.5 windows/amd64
- started: 2026-11-03T03:05:05Z
- ended: 2026-11-03T03:05:09Z
- duration: 4s
- packages: ./pkg/runtime/eventbus/...
- run pattern: ^TestIntegration
- go test exit: 0

## results

| metric | count |
|---|---|
| total tests | 6 |
| PASS | 6 |
| FAIL | 0 |
| SKIP | 0 |
| incomplete | 0 |
| interaction rows | 0 |
| overall | PASS |

## per test

| test | status | elapsed(s) |
|---|---|---|
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisPublishConsumeOrderAndAck | ok | 0.110 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisDuplicateDeliveryHasOneEffect | ok | 0.100 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisClaimRedeliversUnackedEntry | ok | 0.200 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisMalformedEntryGoesToDlq | ok | 0.100 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisStreamTrimIsBounded | ok | 0.150 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisStreamsCarryTTL | ok | 0.150 |

## skipped tests (must be explained)

- none

## failed tests

- none

## interaction coverage (plan §5.5)

| interaction | record file | rows | coverage |
|---|---|---|---|
| test case result | events.jsonl | 6 | yes |
| middleware interactions | interactions.jsonl | 0 | not_applicable: no test in this selection emitted interaction rows |
| user actions | - | 0 | not_applicable: gate script runs no browser code |
| SSE / AG-UI events | - | 0 | not_applicable: no SSE client in this selection |

## redaction

- DSN and Redis address values are replaced with `<redacted-dsn>` / `<redacted-addr>` in all logs.
- Environment values are read only through `ENVIRON` inside awk, so they never reach argv or evidence files.
