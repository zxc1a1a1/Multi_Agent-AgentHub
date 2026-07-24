# Contract Governance

**Status:** Active
**Version:** AgentHub 2.0
**Baseline:** `2.0-initial`

## 1. Purpose

This Contract defines how the AgentHub 2.0 PDR, Contracts, Schema, ADR, implementation, and legacy documents evolve after the initial design batches.

## 2. Precedence

```text
current explicit user decision
-> docs/pdr/AgentHub-2.0-PDR.md
-> project-architecture
-> active domain Contract
-> Schema/OpenAPI/approved ADR
-> implementation and tests
-> legacy material
```

A lower source cannot silently redefine a higher source.

## 3. Change classes

### Editorial

No behavior or boundary change.

Required:

- owning Contract edit;
- document validation.

### Contract defect correction

Clarifies or fixes an inconsistency without changing the approved product direction.

Required:

- owning Contract;
- Schema/OpenAPI when affected;
- regression test/fixture when implemented.

### Domain behavior change

Changes state, validation, lifecycle, API, event, persistence, provider, security, or Artifact semantics within one domain.

Required:

- owning Contract;
- machine-readable contract;
- migration/compatibility;
- tests.

### Cross-module boundary change

Changes service ownership, trust boundary, language boundary, protocol ownership, or introduces a new core domain.

Required:

- ADR;
- affected Contracts;
- Schema/OpenAPI;
- migration/compatibility;
- security and testing review;
- explicit approval.

## 4. Skill governance

A new Skill requires:

- unique non-overlapping responsibility;
- named primary Contract;
- clear trigger conditions;
- no duplication of an existing domain/review Skill;
- retirement or migration plan when replacing a Skill.

Do not create Agent-specific Skills for CodeAgent/WebAgent behavior when AgentCard Skill declarations and Agent implementation documentation are sufficient.

## 5. Versioning

Contract baseline:

```text
2.0-initial
```

A breaking change records:

- affected resources;
- compatibility period;
- migration;
- old reader/writer behavior;
- removal condition.

Historical PlanVersion, AgentVersion, ContextSnapshot, Event, and ArtifactVersion remain interpretable.

## 6. Legacy

Old content moves to `docs/legacy/**`.

Legacy files:

- are read-only historical references by default;
- do not appear in active source precedence;
- are not silently copied back into active Contracts;
- may be removed only under an explicit retention decision.

## 7. Implementation rule

Implementation details do not automatically create new Contracts.

Prefer:

```text
existing Contract section
approved ADR for cross-boundary decision
package-level documentation
tests
```

## 8. Review checklist

- [ ] change class identified.
- [ ] PDR and owning Contract agree.
- [ ] machine-readable contract updated where needed.
- [ ] compatibility and migration are explicit.
- [ ] legacy sources do not override active sources.
- [ ] new Skill/domain is justified and non-overlapping.
- [ ] tests and security review match the change class.
