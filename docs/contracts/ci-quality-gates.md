# CI Quality Gates

## Required baseline jobs

```text
contract-schema
go-test
frontend-test
security-negative
mock-integration
```

## Contract/schema

- parse all JSON/YAML;
- validate JSON Schemas with Draft 2020-12;
- validate OpenAPI;
- check Skill frontmatter/name;
- reject references to retired active Skill names where prohibited.

## Go

```text
go test ./...
go test -race <affected concurrent packages>
go vet <affected packages>
```

Repository-specific workspace commands may replace `./...` when documented.

## Frontend

```text
npm ci
npm run typecheck
npm test -- --run
npm run build
```

Exact scripts follow package.json.

## Security negative

At minimum for affected domains:

- authorization denial;
- redaction;
- SSRF;
- context contamination;
- path traversal;
- preview sandbox policy;
- secret scanning.

## Mock integration

Uses local deterministic services only.

Covers Gateway→Orchestrator→Mock Agent, Plan confirmation, event replay, and Artifact flow.

## Optional jobs

```text
compose-smoke
observability integration
real-provider evaluation
performance
```

Real-provider jobs are opt-in, secret-protected, non-blocking unless explicitly promoted.

## Flake policy

- bounded infrastructure retry only;
- test retry is reported;
- persistent flaky test has owner and removal condition;
- CI does not silently rerun until success.
