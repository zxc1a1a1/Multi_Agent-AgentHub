# Deployment and Demo Profile

**Status:** Active entry document. The default 2.0 Compose profile is a Target while business implementation migrates.

## 2.0 Target

The default Compose profile contains Frontend, Gateway, Orchestrator, CodeAgent and WebAgent. CodeAgent and WebAgent are stable built-in reference Agents; compatible remote A2A Agents register at runtime and do not require a Compose rebuild.

The default persistence target is SQLite/WAL. MySQL and gRPC are not implied dependencies. A mock/deterministic smoke path must not require a production Provider credential.

Direct Mode is a single eligible Agent invocation. Manual Multi-Agent and Auto use an LLM-generated Plan, local validation and user confirmation of the exact PlanVersion before execution.

## Current Implementation

The current `docker-compose.new-arch.yml` is a broader ten-Agent migration topology. Existing `make docker-new-arch-*`, smoke and doctor commands remain usable current implementation commands, but their presence does not prove that every 2.0 Target capability is complete.

Historical deployment guidance belongs under [docs/legacy](../legacy/).
