# Quality Gates

## Contract gate

- active Contract identified;
- Contract/Schema/OpenAPI updated first when boundary changes;
- retired Contract not used as source.

## Build gate

- Go build/test for affected modules;
- frontend typecheck/test;
- JSON Schema/OpenAPI validation;
- generated code current where applicable.

## Behavior gate

- positive path;
- negative path;
- cancellation/error;
- idempotency/concurrency where applicable;
- deterministic Mock path.

## Security gate

- object authorization;
- secret/error redaction;
- registration SSRF when affected;
- context isolation;
- Artifact/preview safety;
- Tool approval.

## Compatibility gate

- migration/adapter documented;
- historical Plan/Agent/Artifact data remains readable;
- deprecated endpoint/event has removal condition;
- rollback described.

## Evidence gate

- commands and results;
- modified/new/deleted files;
- known gaps;
- review verdict.
