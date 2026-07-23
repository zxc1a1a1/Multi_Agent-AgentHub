# A2A AgentCard Contract

**Status:** Active
**Version:** AgentHub 2.0
**Protocol source:** Official A2A specification and official Go SDK

## 1. Purpose

AgentCard describes a remote Agent's identity, supported interfaces, capabilities, security requirements, media modes, and Skills.

Agent Registry owns fetching, validation, versioning, authorization, and health. A2A adapter owns protocol parsing and invocation compatibility.

## 2. Discovery endpoint

Standard target:

```text
GET /.well-known/agent-card.json
```

Legacy compatibility during migration:

```text
GET /.well-known/agent.json
```

Rules:

- new Agents SHOULD publish the standard endpoint;
- Registry tries the configured explicit card URL first;
- legacy fallback is configuration/migration behavior, not a new standard;
- remote dynamic registration fails closed when the required card cannot be validated;
- built-in/config Agents may use trusted local fallback metadata under Registry Contract.

## 3. Wire model

AgentHub uses the official A2A AgentCard model.

Relevant concepts include:

```text
name
description
version
supported interfaces/endpoints
protocol version/binding
capabilities such as streaming
default input/output modes
skills
security schemes/requirements
extensions
```

AgentHub MAY preserve unknown official extension fields but MUST NOT trust them automatically.

AgentHub-specific fields such as visibility, trust level, context tags, or Tool permissions are stored separately in Registry metadata.

## 4. Agent Skill

A Skill declaration includes official fields applicable to the protocol version, such as:

```text
id
name
description
tags
examples
input/output modes
security requirements
```

Rules:

- `skillId` is stable within an AgentVersion;
- Planner may use only locally allowed declared Skills;
- Registry may hide, disable, deprecate, or risk-label a Skill;
- Registry MUST NOT invent a Skill for a remote Agent;
- Skill changes create or record a new AgentVersion.

## 5. Sanitized public projection

Gateway may expose:

```text
agentId
name
description
version
status
health
allowed Skills
input/output modes
streaming
visibility
```

Gateway MUST NOT expose:

```text
credential values
secret headers
private admin endpoints
system prompt
raw internal health errors
private Registry metadata
```

## 6. Security

AgentCard is untrusted remote input.

Validation includes:

- response size/depth limits;
- JSON/type validation;
- endpoint scheme and SSRF policy;
- protocol version support;
- Skill ID uniqueness;
- safe URL resolution;
- no secret persistence in public card fields;
- authentication requirements handled through protected credential storage.

## 7. Version and snapshot

Registry stores:

```text
AgentVersion
card hash
activation/deprecation time
```

Each Run stores the Agent snapshot used for planning/execution.

Historical Runs remain readable after refresh, disable, or removal.

## 8. Compatibility

Current repository custom card structures may remain behind an adapter during migration.

The adapter MUST:

- preserve official semantics;
- surface unsupported fields explicitly;
- not silently rewrite incompatible data;
- have contract tests against official SDK fixtures.

## 9. Required tests

- standard card path;
- configured card URL;
- legacy path compatibility;
- invalid/oversized card rejection;
- duplicate Skill rejection;
- unsupported protocol version;
- sanitized public projection;
- refresh creates/preserves AgentVersion;
- historical snapshot remains after removal.
