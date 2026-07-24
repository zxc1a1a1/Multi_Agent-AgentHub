# Docker Compose Delivery Contract

**Status:** Active
**Version:** AgentHub 2.0

## 1. Purpose

Compose provides a reproducible single-node AgentHub 2.0 development/demo profile.

## 2. Default topology

```text
frontend
  -> gateway
  -> orchestrator
  -> code-agent
  -> web-agent
```

Default built-in Agents:

```text
CodeAgent
WebAgent
```

Other Agents are registered dynamically through Registry and are not mandatory Compose services.

## 3. Data

Default profile:

```text
SQLite
WAL
persistent named volume
single migration owner
```

MySQL, PostgreSQL, Redis, and object storage are optional future/deployment profiles, not default requirements.

## 4. Profiles

### default

- five services;
- deterministic Mock model behavior by default;
- SQLite persistence;
- no production API key.

### observability

Optional collector/backend services configured by environment/profile.

### real-model

May be expressed through environment configuration or explicit profile. Missing required model secret fails clearly without exposing its value.

## 5. Networking

- Frontend public port;
- Gateway public API port;
- Orchestrator and Agent ports internal unless development exposure is explicit;
- service-to-service URLs use Compose DNS names;
- browser never receives internal Agent credential;
- remote Agent URL registration follows Registry security policy.

## 6. Health and readiness

Every service declares readiness meaningful to its role.

```text
frontend      static/dev server available
gateway       API and storage initialized
orchestrator  planner/registry/runtime initialized
Agent         AgentCard and handler ready
```

Startup dependency uses health/readiness where supported, but services also handle dependency delay/retry safely.

## 7. Configuration

Configuration categories:

```text
ports and public origin
service authentication
SQLite path/volume
Planner/model mode
Agent URLs/identity
Registry storage/policy
timeouts and limits
telemetry
preview policy
```

`.env.example` uses placeholders only.

## 8. Secrets

- no secret in Compose file;
- use environment/secret mechanism;
- no browser `VITE_*` secret;
- internal tokens differ from public user token;
- Agent credentials remain protected;
- logs/health never print secret values.

## 9. Mock mode

Mock mode:

- deterministic Planner/Agent output where configured;
- no API key;
- supports streaming fixtures;
- supports plan awaiting-confirmation;
- supports Artifact/WebProject fixture;
- supports controllable failure/cancel cases.

## 10. Dynamic Agents

Remote Agents may run outside Compose.

Registration:

```text
Gateway API
-> Orchestrator Registry
-> remote A2A endpoint
```

Adding a remote Agent does not require editing/rebuilding Compose.

Private-network registration requires an operator-approved network profile.

## 11. Volumes

- SQLite named volume;
- optional attachment/artifact storage volume;
- no source repository mounted read-write into untrusted preview/Agent by default;
- permissions and backup/cleanup documented.

## 12. Shutdown

- Gateway stops accepting new Runs;
- Orchestrator cancels/drains active work according to timeout;
- Agent calls receive cancellation;
- events and terminal state are persisted when possible;
- SQLite closes cleanly.

## 13. Smoke acceptance

Default smoke:

- all readiness endpoints;
- create two Conversations and switch;
- Direct CodeAgent;
- Direct WebAgent;
- Manual or Auto Plan waits for confirmation;
- confirm and execute;
- event replay;
- Artifact creation/version;
- WebProject fixture validity;
- cancel;
- restart persistence;
- log secret scan.

## 14. Required tests

- config validation;
- missing secret in real-model profile;
- volume restart;
- service dependency delay;
- health failure;
- clean shutdown;
- Mock smoke;
- remote Agent registration fixture;
- no public internal ports unless documented;
- no secrets in rendered Compose/logs.
