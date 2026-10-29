# Agent Config Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

Agent YAML config is interpreted by Runtime Agent Factory Registry under `pkg/runtime/registry` and consumed by services under `services/agents/*`.

## Minimal config

```yaml
name: code-agent
description: Code generation agent
version: "0.1.0"
url: http://code-agent:8081
model: claude-sonnet
instruction: |
  You are a code generation assistant.
tools:
  - local/code_tools/generate_code
skills:
  - code_generation
inputModes: [text]
outputModes: [text, code]
streaming: true
max_iterations: 25
context_pruning:
  strategy: keep_ends_window
  keep_first: 3
  keep_last: 20
  max_tool_result_len: 4096
```

## Rules

- Config must not include raw secrets. Use env var references.
- `model` references Runtime model registry.
- `tools` use `type/set/func` paths.
- `skills` reference Runtime Skill repository names.

## v2 tool and memory config

```yaml
tools:
  - name: workspace.read
    type: native/workspace/read
    readOnly: true
    timeoutMs: 15000
    maxOutputBytes: 65536
  - name: workspace.apply_patch
    type: native/workspace/apply_patch
    readOnly: false
    approval: always
memory:
  mode: session_only        # off | session_only | scoped_long_term
  scope: derived_from_request
budget:
  modelCalls: 16
  toolCalls: 32
  executionSeconds: 180
```

- 工具声明只描述能力，不构成授权；实际调用仍经 Tool Broker。
- `approval` 取 `none | confirm | always`；写工具默认 `always`。
- Memory `mode` 默认 `session_only`；`off` 不删除已有聊天/checkpoint/审计。
- 配置不能包含密钥；缺失模型配置必须启动/就绪失败，不得回退规则模式。
- 子 Agent 默认不继承父级全部工具、全部历史或继续委派权限。
