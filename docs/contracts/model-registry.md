# Model Registry Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Shape

```yaml
models:
  - name: claude-sonnet
    provider: anthropic
    api_key_env: ANTHROPIC_API_KEY
    model: claude-sonnet-4-20250514
    base_url: https://api.anthropic.com
```

## Rules

- `provider` maps to a registered Runtime `ModelProvider`.
- `name` is the stable reference used by Agent YAML config.
- No model instance is constructed before config/env validation.
