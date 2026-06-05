# Storage Profiles Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Profiles

| Profile | Use | Status |
|---|---|---|
| `mysql` | Full redesign target for Gateway and Runtime session persistence | target |
| `sqlite-demo` | Local/demo compatibility only | temporary |
| `memory` | Unit tests and ephemeral child-agent runtime | active for tests |
| `object-storage` | Large file artifacts | future |

## Rule

Contracts must label non-target profiles explicitly. Do not silently replace MySQL target with SQLite in delivery docs.
