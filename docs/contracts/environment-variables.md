# Environment Variable Contract

## Rules

- names are uppercase `SNAKE_CASE`;
- `.env.example` contains placeholders only;
- secrets have no insecure production default;
- services validate required combinations at startup;
- unknown/deprecated variables generate a safe warning where practical;
- values are never printed in full.

## Gateway

```text
GATEWAY_ADDR
GATEWAY_ALLOWED_ORIGINS
GATEWAY_ENABLE_AUTH
AGENTHUB_API_TOKEN
ORCHESTRATOR_URL
ORCHESTRATOR_INTERNAL_TOKEN
AGENTHUB_GATEWAY_STORE
AGENTHUB_SQLITE_PATH
```

SQLite is the default 2.0 profile.

## Orchestrator

```text
ORCHESTRATOR_ADDR
INTERNAL_SERVICE_TOKEN
ORCHESTRATOR_PLANNER_MODE
ORCHESTRATOR_LLM_PROVIDER
ORCHESTRATOR_LLM_MODEL
ORCHESTRATOR_LLM_API_KEY
ORCHESTRATOR_LLM_BASE_URL
REGISTRY_STORE
REGISTRY_DATA_PATH
REGISTRY_HEALTHCHECK
ORCHESTRATOR_MAX_TASKS
DISPATCH_TIMEOUT_MS
DISPATCH_MAX_RETRY
DISPATCH_RETRY_BACKOFF_MS
```

Planner mode defaults must align with active Planning Contract. Mock/CI mode is explicit.

## Built-in Agents

Common pattern:

```text
{PREFIX}_AGENT_ADDR
{PREFIX}_AGENT_PUBLIC_URL
{PREFIX}_AGENT_VERSION
{PREFIX}_AGENT_MODE
{PREFIX}_AGENT_LLM_PROVIDER
{PREFIX}_AGENT_LLM_MODEL
{PREFIX}_AGENT_LLM_API_KEY
{PREFIX}_AGENT_LLM_BASE_URL
{PREFIX}_AGENT_TIMEOUT_MS
```

Stable prefixes:

```text
CODE
WEB
```

Other registered remote Agents do not require a predefined environment-variable prefix.

## Preview

```text
VITE_AGENTHUB_API_BASE
VITE_PREVIEW_ENABLED
VITE_PREVIEW_NETWORK_POLICY
```

Do not put API keys or Agent credentials in `VITE_*`.

## Observability

```text
OTEL_SERVICE_NAME
OTEL_EXPORTER_OTLP_ENDPOINT
OTEL_TRACES_EXPORTER
OTEL_METRICS_EXPORTER
OTEL_RESOURCE_ATTRIBUTES
```

Telemetry endpoint is configuration, not a user-visible secret.

## Deprecated variables

Variables for old mandatory MySQL/gRPC/ten-Agent profiles may remain temporarily with documented adapter/removal conditions. They are not the active 2.0 default.
