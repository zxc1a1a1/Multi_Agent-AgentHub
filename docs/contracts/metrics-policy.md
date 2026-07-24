# Metrics Policy

## 1. Goals

Metrics answer:

- whether product flows succeed;
- where latency and failures occur;
- whether context/token/cost budgets are controlled;
- whether Agent, Tool, Artifact, and Preview subsystems are healthy.

## 2. Cardinality

Allowed labels are bounded enums/configured registries such as:

```text
service
environment
mode
status
result
error_class
use_case
provider
configured model
agent source/trust class
artifact type
preview template
```

Do not use as unrestricted metric labels:

```text
userId
conversationId
messageId
runId
planId
invocationId
dynamic agentId
artifactId
URL
free-form error text
```

Use traces/logs for exact IDs.

## 3. Privacy

Metrics never contain:

- prompt/content;
- credentials;
- email/name;
- Attachment/Artifact body;
- source URL unless classified into a bounded domain policy;
- private model reasoning.

## 4. Required groups

```text
run and plan
context and retrieval
Agent/A2A
model/token/cost
Tool
persistence/event replay
Artifact/version conflict
preview
Registry health
```

## 5. SLO candidates

Measure, without claiming unsupported production guarantees:

- Run success and latency;
- Plan valid/confirmation rate;
- event replay correctness;
- Agent invocation success;
- context budget overflow;
- Artifact conflict;
- Preview startup;
- deterministic smoke pass.

SLO values are set only after a measured baseline.

## 6. Review

Every new metric documents:

```text
name
type
unit
description
labels and cardinality
owner
privacy classification
retention
alert/dashboard use
```
