# 审核交接说明

执行 agent 完成后，把以下内容交给审核方。

## 必须提供

```bash
git diff --stat
git diff --name-only
```

建议也提供：

```bash
git diff
```

## 必须提供的报告

```text
Phase 0 Report
Phase 1 Report
Phase 2 Report
Phase 3 Report
Phase 4 Report
Phase 5 Report
Final Report
```

## 必须提供的测试证据

```text
Parser tests
Normalizer tests
Validator tests
LLMPlanner valid tests
Repair success tests
Repair fail fallback tests
Metadata/wiring tests
```

## 审核启动语

```text
/llm-orchestration-review

请按审核 skill 检查这次 Orchestrator LLM 编排开发。
重点检查路径边界、LLMPlanner 主路径、Validator、Repairer、RulePlanner 是否被增强、前端事件兼容、测试证据。
```
