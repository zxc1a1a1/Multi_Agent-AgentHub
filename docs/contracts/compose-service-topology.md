# Compose Service Topology

## Default

```text
Browser
  -> frontend:3000
  -> gateway:8080
     -> orchestrator:8090
        -> code-agent:8081
        -> web-agent:8082
```

Exact ports may be configured, but service identity and direction remain.

## Exposure

Public/development host exposure:

```text
frontend
gateway
```

Internal by default:

```text
orchestrator
code-agent
web-agent
optional telemetry backends
```

Development may expose internal ports explicitly for debugging.

## Dynamic Agent

A remote registered Agent is outside the default service list:

```text
orchestrator
-> validated registered endpoint
```

It is not added to Compose automatically.

## Storage

```text
gateway/orchestrator repository layer
-> SQLite volume
```

Deployment must follow the single-writer/repository ownership policy and WAL settings defined by Data Persistence Contract.

## Optional observability

```text
services
-> OTLP collector
-> selected local backend
```

Observability services are activated through a profile and are not required for core startup.
