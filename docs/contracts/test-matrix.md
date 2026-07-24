# AgentHub 2.0 Test Matrix

| Domain | Contract | Unit | Integration | Negative/Security | Persistence/Replay |
|---|---|---|---|---|---|
| Conversation | CRUD, modes, Message order | reducers/services | Gateway + storage | cross-Conversation access | restart, pagination |
| Planning | PlanVersion and validation | parser/validator | Planner + confirmation | unconfirmed, stale, cycle | version history |
| Registry | AgentCard/version/health | normalization/policy | local mock A2A | SSRF, credential, authorization | Agent snapshot |
| Context | projection and budget | selector/budget | summary + FTS | contamination, secret redaction | ContextSnapshot |
| A2A | Task/stream mapping | adapters | local mock Agent | cancel, protocol error | event ordering |
| Artifact | immutable versions | patch/type | Agent output + storage | traversal, unknown type | restore/purge |
| Preview | WebProject schema | mapping/reducer | Sandpack fixture | sandbox/token/network | version restore |
| Compose | topology/config | validation | deterministic smoke | secret/port checks | volume restart |

## Required cross-domain scenarios

```text
Direct CodeAgent
Direct WebAgent
Direct dynamically registered Mock Agent
Manual Plan awaiting confirmation
Auto Plan from eligible Registry Catalog
confirmed execution
material Replan and reconfirmation
event reconnect and replay
Artifact creation and version conflict
Conversation switch without context leakage
```

Every implementation batch marks affected rows and reports the exact evidence.
